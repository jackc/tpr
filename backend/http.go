package backend

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/netip"
	"net/url"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tpr/backend/data"
	log "gopkg.in/inconshreveable/log15.v2"
)

type HTTPConfig struct {
	ListenAddress string
	ListenPort    string
	StaticURL     string
}

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

type AppServer struct {
	handler    http.Handler
	httpConfig HTTPConfig
	server     *http.Server
}

func NewAppServer(httpConfig HTTPConfig, pool *pgxpool.Pool, mailer Mailer, logger log.Logger) (*AppServer, error) {
	r := chi.NewRouter()

	if httpConfig.StaticURL != "" {
		staticURL, err := url.Parse(httpConfig.StaticURL)
		if err != nil {
			return nil, fmt.Errorf("bad static-url: %v", err)
		}
		r.Handle("/*", httputil.NewSingleHostReverseProxy(staticURL))
	}

	apiHandler := NewAPIHandler(pool, mailer, logger.New("module", "http"))
	r.Mount("/api", apiHandler)

	return &AppServer{
		handler:    r,
		httpConfig: httpConfig,
	}, nil
}

func (s *AppServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

func (s *AppServer) Serve() error {
	listenAt := fmt.Sprintf("%s:%s", s.httpConfig.ListenAddress, s.httpConfig.ListenPort)
	s.server = &http.Server{
		Addr:    listenAt,
		Handler: s.handler,
	}

	fmt.Printf("Starting to listen on: %s\n", listenAt)

	err := s.server.ListenAndServe()
	if err != http.ErrServerClosed {
		return err
	}

	return nil
}

func (s *AppServer) Shutdown(ctx context.Context) error {
	s.server.SetKeepAlivesEnabled(false)
	err := s.server.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("graceful HTTP server shutdown failed: %w", err)
	}

	return nil
}

type environment struct {
	user   *data.User
	pool   *pgxpool.Pool
	logger log.Logger
	mailer Mailer
}

func NewAPIHandler(pool *pgxpool.Pool, mailer Mailer, logger log.Logger) chi.Router {
	router := chi.NewRouter()

	router.Method("POST", "/register", EnvHandler(pool, mailer, logger, RegisterHandler))
	router.Method("POST", "/sessions", EnvHandler(pool, mailer, logger, CreateSessionHandler))
	router.Method("DELETE", "/sessions/{id}", EnvHandler(pool, mailer, logger, AuthenticatedHandler(DeleteSessionHandler)))
	router.Method("POST", "/subscriptions", EnvHandler(pool, mailer, logger, AuthenticatedHandler(CreateSubscriptionHandler)))
	router.Method("DELETE", "/subscriptions/{id}", EnvHandler(pool, mailer, logger, AuthenticatedHandler(DeleteSubscriptionHandler)))
	router.Method("POST", "/request_password_reset", EnvHandler(pool, mailer, logger, RequestPasswordResetHandler))
	router.Method("POST", "/reset_password", EnvHandler(pool, mailer, logger, ResetPasswordHandler))
	router.Method("GET", "/feeds", EnvHandler(pool, mailer, logger, AuthenticatedHandler(GetFeedsHandler)))
	router.Method("POST", "/feeds/import", EnvHandler(pool, mailer, logger, AuthenticatedHandler(ImportFeedsHandler)))
	router.Method("GET", "/feeds.xml", EnvHandler(pool, mailer, logger, AuthenticatedHandler(ExportFeedsHandler)))
	router.Method("GET", "/items/unread", EnvHandler(pool, mailer, logger, AuthenticatedHandler(GetUnreadItemsHandler)))
	router.Method("POST", "/items/unread/mark_multiple_read", EnvHandler(pool, mailer, logger, AuthenticatedHandler(MarkMultipleItemsReadHandler)))
	router.Method("DELETE", "/items/unread/{id}", EnvHandler(pool, mailer, logger, AuthenticatedHandler(MarkItemReadHandler)))
	router.Method("GET", "/items/archived", EnvHandler(pool, mailer, logger, AuthenticatedHandler(GetArchivedItemsHandler)))
	router.Method("GET", "/account", EnvHandler(pool, mailer, logger, AuthenticatedHandler(GetAccountHandler)))
	router.Method("PATCH", "/account", EnvHandler(pool, mailer, logger, AuthenticatedHandler(UpdateAccountHandler)))

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

func newStringFallback(value string) pgtype.Text {
	if value == "" {
		return pgtype.Text{}
	} else {
		return pgtype.Text{String: value, Valid: true}
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
	user.Name = pgtype.Text{String: registration.Name, Valid: true}
	user.Email = newStringFallback(registration.Email)
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
			ID:     sessionID,
			UserID: userID,
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

	if err := data.InsertSubscription(context.Background(), env.pool, env.user.ID.Int32, subscription.URL); err != nil {
		w.WriteHeader(422)
		fmt.Fprintln(w, `Bad user name or password`)
		return
	}

	w.WriteHeader(http.StatusCreated)
}

func DeleteSubscriptionHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	feedID, err := strconv.ParseInt(chi.URLParam(req, "id"), 10, 32)
	if err != nil {
		// If not an integer it clearly can't be found
		http.NotFound(w, req)
		return
	}

	if err := data.DeleteSubscription(context.Background(), env.pool, env.user.ID.Int32, int32(feedID)); err != nil {
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
			ID:     sessionID,
			UserID: user.ID.Int32,
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
	sessionID, err := hex.DecodeString(chi.URLParam(req, "id"))
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
	if err := data.CopyUnreadItemsAsJSONByUserID(context.Background(), env.pool, w, env.user.ID.Int32); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func MarkItemReadHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	itemID, err := strconv.ParseInt(chi.URLParam(req, "id"), 10, 32)
	if err != nil {
		// If not an integer it clearly can't be found
		http.NotFound(w, req)
		return
	}

	err = data.MarkItemRead(context.Background(), env.pool, env.user.ID.Int32, int32(itemID))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			http.NotFound(w, req)
			return
		}
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
		err := data.MarkItemRead(context.Background(), env.pool, env.user.ID.Int32, itemID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
	}
}

func GetArchivedItemsHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	w.Header().Set("Content-Type", "application/json")
	if err := data.CopyArchivedItemsAsJSONByUserID(context.Background(), env.pool, w, env.user.ID.Int32); err != nil {
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
			err := data.InsertSubscription(context.Background(), env.pool, env.user.ID.Int32, outline.URL)
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
	subs, err := data.SelectSubscriptions(context.Background(), env.pool, env.user.ID.Int32)
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
	if err := data.CopySubscriptionsForUserAsJSON(context.Background(), env.pool, w, env.user.ID.Int32); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func GetAccountHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	var user struct {
		ID    int32  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	user.ID = env.user.ID.Int32
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

	user, err := data.SelectUserByPK(context.Background(), env.pool, env.user.ID.Int32)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, `Internal server error`)
		env.logger.Error("SelectUserByPK", "err", err)
	}

	user.Email = newStringFallback(update.Email)

	if update.NewPassword != "" {
		err := SetPassword(user, update.NewPassword)
		if err != nil {
			w.WriteHeader(422)
			fmt.Fprintln(w, err)
			return
		}
	}

	err = data.UpdateUser(context.Background(), env.pool, env.user.ID.Int32, user)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, `Internal server error`)
		env.logger.Error("UpdateUser", "err", err)
	}
}

func RequestPasswordResetHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	pwr := data.PasswordResetsTable.NewRecord()
	pwr.MustSet("request_time", time.Now())

	if host, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		if addr, err := netip.ParseAddr(host); err == nil {
			pwr.MustSet("request_ip", addr)
		}
	}

	token, err := genLostPasswordToken()
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, `Internal server error`)
		env.logger.Error("getLostPasswordToken failed", "error", err)
		return
	}
	pwr.MustSet("token", token)

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

	pwr.MustSet("email", reset.Email)

	user, err := data.SelectUserByEmail(context.Background(), env.pool, reset.Email)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(500)
			fmt.Fprintln(w, `Internal server error`)
			return
		}
	}

	if user != nil {
		pwr.MustSet("user_id", user.ID)
	}

	err = pwr.Save(context.Background(), env.pool)
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
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			w.WriteHeader(404)
			return
		}

		w.WriteHeader(500)
		fmt.Fprintf(w, "Error decoding request: %v", err)
		return
	}

	if !pwr.UserID.Valid {
		w.WriteHeader(404)
		return
	}

	if pwr.CompletionTime.Valid {
		w.WriteHeader(404)
		return
	}

	attrs, err := data.SelectUserByPK(context.Background(), env.pool, pwr.UserID.Int32)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, `Internal server error`)
		env.logger.Error("SelectUserByPK", "err", err)
	}
	SetPassword(attrs, resetPassword.Password)

	err = data.UpdateUser(context.Background(), env.pool, pwr.UserID.Int32, attrs)
	if err != nil {
		w.WriteHeader(500)
		return
	}

	user, err := data.SelectUserByPK(context.Background(), env.pool, pwr.UserID.Int32)
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
			ID:     sessionID,
			UserID: user.ID.Int32,
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
