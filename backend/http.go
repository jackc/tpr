package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net"
	"net/http"
	"strconv"
	"time"

	"github.com/jackc/pgtype"
	pgxpool "github.com/jackc/pgx/v4/pool"
	qv "github.com/jackc/quo_vadis"
	"github.com/jackc/tpr/backend/data"
	log "gopkg.in/inconshreveable/log15.v2"
)

type EnvHandlerFunc func(w http.ResponseWriter, req *http.Request, env *environment)

func EnvHandler(pool *pgxpool.Pool, mailer Mailer, logger log.Logger, f EnvHandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user := getUserFromSession(req, pool)
		env := &environment{user: user, pool: pool, mailer: mailer, logger: logger}
		f(w, req, env)
	})
}

func AuthenticatedHandler(f EnvHandlerFunc) EnvHandlerFunc {
	return EnvHandlerFunc(func(w http.ResponseWriter, req *http.Request, env *environment) {
		if env.user == nil {
			w.WriteHeader(http.StatusForbidden)
			fmt.Fprint(w, "Bad or missing X-Authentication header")
			return
		}
		f(w, req, env)
	})
}

type environment struct {
	user   *data.User
	pool   *pgxpool.Pool
	logger log.Logger
	mailer Mailer
}

func NewAPIHandler(pool *pgxpool.Pool, mailer Mailer, logger log.Logger) http.Handler {
	router := qv.NewRouter()

	router.Post("/register", EnvHandler(pool, mailer, logger, RegisterHandler))
	router.Post("/sessions", EnvHandler(pool, mailer, logger, CreateSessionHandler))
	router.Delete("/sessions/:id", EnvHandler(pool, mailer, logger, AuthenticatedHandler(DeleteSessionHandler)))
	router.Post("/subscriptions", EnvHandler(pool, mailer, logger, AuthenticatedHandler(CreateSubscriptionHandler)))
	router.Delete("/subscriptions/:id", EnvHandler(pool, mailer, logger, AuthenticatedHandler(DeleteSubscriptionHandler)))
	router.Post("/request_password_reset", EnvHandler(pool, mailer, logger, RequestPasswordResetHandler))
	router.Post("/reset_password", EnvHandler(pool, mailer, logger, ResetPasswordHandler))
	router.Get("/feeds", EnvHandler(pool, mailer, logger, AuthenticatedHandler(GetFeedsHandler)))
	router.Post("/feeds/import", EnvHandler(pool, mailer, logger, AuthenticatedHandler(ImportFeedsHandler)))
	router.Get("/feeds.xml", EnvHandler(pool, mailer, logger, AuthenticatedHandler(ExportFeedsHandler)))
	router.Get("/items/unread", EnvHandler(pool, mailer, logger, AuthenticatedHandler(GetUnreadItemsHandler)))
	router.Post("/items/unread/mark_multiple_read", EnvHandler(pool, mailer, logger, AuthenticatedHandler(MarkMultipleItemsReadHandler)))
	router.Delete("/items/unread/:id", EnvHandler(pool, mailer, logger, AuthenticatedHandler(MarkItemReadHandler)))
	router.Get("/items/archived", EnvHandler(pool, mailer, logger, AuthenticatedHandler(GetArchivedItemsHandler)))
	router.Get("/account", EnvHandler(pool, mailer, logger, AuthenticatedHandler(GetAccountHandler)))
	router.Patch("/account", EnvHandler(pool, mailer, logger, AuthenticatedHandler(UpdateAccountHandler)))

	return router
}

func getUserFromSession(req *http.Request, pool *pgxpool.Pool) *data.User {
	token := req.Header.Get("X-Authentication")
	if token == "" {
		token = req.FormValue("session")
	}

	var sessionID []byte
	sessionID, err := hex.DecodeString(token)
	if err != nil {
		return nil
	}

	// TODO - this could be an error from no records found -- or the connection could be dead or we could have a syntax error...
	user, err := data.SelectUserBySessionID(context.Background(), pool, sessionID)
	if err != nil {
		return nil
	}

	return user
}

func newStringFallback(value string, status pgtype.Status) pgtype.Varchar {
	if value == "" {
		return pgtype.Varchar{Status: status}
	} else {
		return pgtype.Varchar{String: value, Status: pgtype.Present}
	}
}

func RegisterHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	var registration struct {
		Name     string `json:"name"`
		Email    string `json:"email"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&registration); err != nil {
		w.WriteHeader(422)
		fmt.Fprintf(w, "Error decoding request: %v", err)
		return
	}

	if registration.Name == "" {
		w.WriteHeader(422)
		fmt.Fprintln(w, `Request must include the attribute "name"`)
		return
	}

	if len(registration.Name) > 30 {
		w.WriteHeader(422)
		fmt.Fprintln(w, `"name" must be less than 30 characters`)
		return
	}

	err := validatePassword(registration.Password)
	if err != nil {
		w.WriteHeader(422)
		fmt.Fprintln(w, err)
		return
	}

	user := &data.User{}
	user.Name = pgtype.Varchar{String: registration.Name, Status: pgtype.Present}
	user.Email = newStringFallback(registration.Email, pgtype.Undefined)
	SetPassword(user, registration.Password)

	userID, err := data.CreateUser(context.Background(), env.pool, user)
	if err != nil {
		if err, ok := err.(data.DuplicationError); ok {
			w.WriteHeader(422)
			fmt.Fprintf(w, `"%s" is already taken`, err.Field)
			return
		} else {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
	}

	sessionID, err := genSessionID()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = data.InsertSession(context.Background(),
		env.pool,
		&data.Session{
			ID:     pgtype.Bytea{Bytes: sessionID, Status: pgtype.Present},
			UserID: pgtype.Int4{Int: userID, Status: pgtype.Present},
		},
	)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	var response struct {
		Name      string `json:"name"`
		SessionID string `json:"sessionID"`
	}

	response.Name = registration.Name
	response.SessionID = hex.EncodeToString(sessionID)

	encoder := json.NewEncoder(w)
	encoder.Encode(response)
}

func CreateSubscriptionHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	var subscription struct {
		URL string `json:"url"`
	}

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&subscription); err != nil {
		w.WriteHeader(422)
		fmt.Fprintf(w, "Error decoding request: %v", err)
		return
	}

	if subscription.URL == "" {
		w.WriteHeader(422)
		fmt.Fprintln(w, `Request must include the attribute "url"`)
		return
	}

	if err := data.InsertSubscription(context.Background(), env.pool, env.user.ID.Int, subscription.URL); err != nil {
		w.WriteHeader(422)
		fmt.Fprintln(w, `Bad user name or password`)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func DeleteSubscriptionHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	feedID, err := strconv.ParseInt(req.FormValue("id"), 10, 32)
	if err != nil {
		// If not an integer it clearly can't be found
		http.NotFound(w, req)
		return
	}

	if err := data.DeleteSubscription(context.Background(), env.pool, env.user.ID.Int, int32(feedID)); err != nil {
		w.WriteHeader(422)
		fmt.Fprintf(w, "Error deleting subscription: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func CreateSessionHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	var credentials struct {
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&credentials); err != nil {
		w.WriteHeader(422)
		fmt.Fprintf(w, "Error decoding request: %v", err)
		return
	}

	if credentials.Name == "" {
		w.WriteHeader(422)
		fmt.Fprintln(w, `Request must include the attribute "name"`)
		return
	}

	if credentials.Password == "" {
		w.WriteHeader(422)
		fmt.Fprintln(w, `Request must include the attribute "password"`)
		return
	}

	user, err := data.SelectUserByName(context.Background(), env.pool, credentials.Name)
	if err != nil {
		w.WriteHeader(422)
		fmt.Fprintln(w, "Bad user name or password")
		return
	}

	if !IsPassword(user, credentials.Password) {
		w.WriteHeader(422)
		fmt.Fprintln(w, "Bad user name or password")
		return
	}

	sessionID, err := genSessionID()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = data.InsertSession(context.Background(),
		env.pool,
		&data.Session{
			ID:     pgtype.Bytea{Bytes: sessionID, Status: pgtype.Present},
			UserID: user.ID,
		},
	)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	var response struct {
		Name      string `json:"name"`
		SessionID string `json:"sessionID"`
	}

	response.Name = credentials.Name
	response.SessionID = hex.EncodeToString(sessionID)

	encoder := json.NewEncoder(w)
	encoder.Encode(response)

}

func DeleteSessionHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	sessionID, err := hex.DecodeString(req.FormValue("id"))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	err = data.DeleteSession(context.Background(), env.pool, sessionID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{Name: "sessionId", Value: "logged out", Expires: time.Unix(0, 0)}
	http.SetCookie(w, cookie)
}

func GetUnreadItemsHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	w.Header().Set("Content-Type", "application/json")
	if err := data.CopyUnreadItemsAsJSONByUserID(context.Background(), env.pool, w, env.user.ID.Int); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func MarkItemReadHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	itemID, err := strconv.ParseInt(req.FormValue("id"), 10, 32)
	if err != nil {
		// If not an integer it clearly can't be found
		http.NotFound(w, req)
		return
	}

	err = data.MarkItemRead(context.Background(), env.pool, env.user.ID.Int, int32(itemID))
	if err == data.ErrNotFound {
		http.NotFound(w, req)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func MarkMultipleItemsReadHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	var request struct {
		ItemIDs []int32 `json:"itemIDs"`
	}

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&request); err != nil {
		w.WriteHeader(422)
		fmt.Fprintf(w, "Error decoding request: %v", err)
		return
	}

	for _, itemID := range request.ItemIDs {
		err := data.MarkItemRead(context.Background(), env.pool, env.user.ID.Int, itemID)
		if err != nil && err != data.ErrNotFound {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

func GetArchivedItemsHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	w.Header().Set("Content-Type", "application/json")
	if err := data.CopyArchivedItemsAsJSONByUserID(context.Background(), env.pool, w, env.user.ID.Int); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func ImportFeedsHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	file, _, err := req.FormFile("file")
	if err != nil {
		w.WriteHeader(422)
		fmt.Fprintln(w, `No uploaded file found`)
		return
	}
	defer file.Close()

	var doc OpmlDocument
	err = xml.NewDecoder(file).Decode(&doc)
	if err != nil {
		w.WriteHeader(422)
		fmt.Fprintln(w, "Error parsing OPML upload")
		return
	}

	type subscriptionResult struct {
		Title   string `json:"title"`
		URL     string `json:"url"`
		Success bool   `json:"success"`
	}

	results := make([]subscriptionResult, 0, len(doc.Body.Outlines))
	resultsChan := make(chan subscriptionResult)

	for _, outline := range doc.Body.Outlines {
		go func(outline OpmlOutline) {
			r := subscriptionResult{Title: outline.Title, URL: outline.URL}
			err := data.InsertSubscription(context.Background(), env.pool, env.user.ID.Int, outline.URL)
			r.Success = err == nil
			resultsChan <- r
		}(outline)
	}

	for _ = range doc.Body.Outlines {
		r := <-resultsChan
		results = append(results, r)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func ExportFeedsHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	subs, err := data.SelectSubscriptions(context.Background(), env.pool, env.user.ID.Int)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	doc := OpmlDocument{Version: "1.0"}
	doc.Head.Title = "The Pithy Reader Export for " + env.user.Name.String

	for _, s := range subs {
		doc.Body.Outlines = append(doc.Body.Outlines, OpmlOutline{
			Text:  s.Name.String,
			Title: s.Name.String,
			Type:  "rss",
			URL:   s.URL.String,
		})
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", `attachment; filename="opml.xml"`)
	fmt.Fprint(w, xml.Header)
	xml.NewEncoder(w).Encode(doc)
}

func GetFeedsHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	w.Header().Set("Content-Type", "application/json")
	if err := data.CopySubscriptionsForUserAsJSON(context.Background(), env.pool, w, env.user.ID.Int); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func GetAccountHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	var user struct {
		ID    int32  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	user.ID = env.user.ID.Int
	user.Name = env.user.Name.String
	user.Email = env.user.Email.String

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(user)
}

func UpdateAccountHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	var update struct {
		Email            string `json:"email"`
		ExistingPassword string `json:"existingPassword"`
		NewPassword      string `json:"newPassword"`
	}

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&update); err != nil {
		w.WriteHeader(422)
		fmt.Fprintf(w, "Error decoding request: %v", err)
		return
	}

	if !IsPassword(env.user, update.ExistingPassword) {
		w.WriteHeader(422)
		fmt.Fprintln(w, "Bad existing password")
		return
	}

	user := &data.User{}
	user.Email = newStringFallback(update.Email, pgtype.Null)

	if update.NewPassword != "" {
		err := SetPassword(user, update.NewPassword)
		if err != nil {
			w.WriteHeader(422)
			fmt.Fprintln(w, err)
			return
		}
	}

	err := data.UpdateUser(context.Background(), env.pool, env.user.ID.Int, user)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, `Internal server error`)
		env.logger.Error("UpdateUser", "err", err)
	}
}

func RequestPasswordResetHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	pwr := &data.PasswordReset{}
	pwr.RequestTime = pgtype.Timestamptz{Time: time.Now(), Status: pgtype.Present}

	if host, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		if ip := net.ParseIP(host); ip != nil {
			mask := net.CIDRMask(len(ip)*8, len(ip)*8)
			pwr.RequestIP = pgtype.Inet{IPNet: &net.IPNet{IP: ip, Mask: mask}, Status: pgtype.Present}
		}
	}

	token, err := genLostPasswordToken()
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, `Internal server error`)
		env.logger.Error("getLostPasswordToken failed", "error", err)
		return
	}
	pwr.Token = pgtype.Varchar{String: token, Status: pgtype.Present}

	var reset struct {
		Email string `json:"email"`
	}

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&reset); err != nil {
		w.WriteHeader(422)
		fmt.Fprintf(w, "Error decoding request: %v", err)
		return
	}
	if reset.Email == "" {
		w.WriteHeader(422)
		fmt.Fprint(w, "Error decoding request: missing email")
		return
	}

	pwr.Email = pgtype.Varchar{String: reset.Email, Status: pgtype.Present}

	user, err := data.SelectUserByEmail(context.Background(), env.pool, reset.Email)
	switch err {
	case nil:
		pwr.UserID = user.ID
	case data.ErrNotFound:
	default:
		w.WriteHeader(500)
		fmt.Fprintln(w, `Internal server error`)
		return
	}

	err = data.InsertPasswordReset(context.Background(), env.pool, pwr)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, `Internal server error`)
		env.logger.Error("repo.CreatePasswordReset failed", "error", err)
		return
	}

	if user == nil {
		env.logger.Warn("Password reset requested for missing email", "email", reset.Email)
		return
	}

	if env.mailer == nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, `Internal server error`)
		env.logger.Error("Mail is not configured -- cannot send password reset email")
		return
	}

	err = env.mailer.SendPasswordResetMail(reset.Email, token)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, `Internal server error`)
		env.logger.Error("env.mailer.SendPasswordResetMail failed", "error", err)
		return
	}
}

func ResetPasswordHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	var resetPassword struct {
		Token    string `json:"token"`
		Password string `json:"password"`
	}

	decoder := json.NewDecoder(req.Body)
	if err := decoder.Decode(&resetPassword); err != nil {
		w.WriteHeader(422)
		fmt.Fprintf(w, "Error decoding request: %v", err)
		return
	}

	pwr, err := data.SelectPasswordResetByPK(context.Background(), env.pool, resetPassword.Token)
	if err == data.ErrNotFound {
		w.WriteHeader(404)
		return
	} else if err != nil {
		w.WriteHeader(500)
		fmt.Fprintf(w, "Error decoding request: %v", err)
		return
	}

	if pwr.UserID.Status != pgtype.Present {
		w.WriteHeader(404)
		return
	}

	if pwr.CompletionTime.Status == pgtype.Present {
		w.WriteHeader(404)
		return
	}

	attrs := &data.User{}
	SetPassword(attrs, resetPassword.Password)

	err = data.UpdateUser(context.Background(), env.pool, pwr.UserID.Int, attrs)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	user, err := data.SelectUserByPK(context.Background(), env.pool, pwr.UserID.Int)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	sessionID, err := genSessionID()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = data.InsertSession(context.Background(),
		env.pool,
		&data.Session{
			ID:     pgtype.Bytea{Bytes: sessionID, Status: pgtype.Present},
			UserID: user.ID,
		},
	)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	var response struct {
		Name      string `json:"name"`
		SessionID string `json:"sessionID"`
	}

	response.Name = user.Name.String
	response.SessionID = hex.EncodeToString(sessionID)

	encoder := json.NewEncoder(w)
	encoder.Encode(response)
}
