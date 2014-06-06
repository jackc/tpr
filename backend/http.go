package main

import (
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"github.com/JackC/box"
	qv "github.com/JackC/quo_vadis"
	log "gopkg.in/inconshreveable/log15.v2"
	"net/http"
	"strconv"
	"time"
)

type EnvHandlerFunc func(w http.ResponseWriter, req *http.Request, env *environment)

func EnvHandler(repo repository, logger log.Logger, f EnvHandlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user := getUserFromSession(req, repo)
		env := &environment{user: user, repo: repo, logger: logger}
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
	user   *User
	repo   repository
	logger log.Logger
}

func NewAPIHandler(repo repository, logger log.Logger) http.Handler {
	router := qv.NewRouter()

	router.Post("/register", EnvHandler(repo, logger, RegisterHandler))
	router.Post("/sessions", EnvHandler(repo, logger, CreateSessionHandler))
	router.Delete("/sessions/:id", EnvHandler(repo, logger, AuthenticatedHandler(DeleteSessionHandler)))
	router.Post("/subscriptions", EnvHandler(repo, logger, AuthenticatedHandler(CreateSubscriptionHandler)))
	router.Delete("/subscriptions/:id", EnvHandler(repo, logger, AuthenticatedHandler(DeleteSubscriptionHandler)))
	router.Get("/feeds", EnvHandler(repo, logger, AuthenticatedHandler(GetFeedsHandler)))
	router.Post("/feeds/import", EnvHandler(repo, logger, AuthenticatedHandler(ImportFeedsHandler)))
	router.Get("/feeds.xml", EnvHandler(repo, logger, AuthenticatedHandler(ExportFeedsHandler)))
	router.Get("/items/unread", EnvHandler(repo, logger, AuthenticatedHandler(GetUnreadItemsHandler)))
	router.Post("/items/unread/mark_multiple_read", EnvHandler(repo, logger, AuthenticatedHandler(MarkMultipleItemsReadHandler)))
	router.Delete("/items/unread/:id", EnvHandler(repo, logger, AuthenticatedHandler(MarkItemReadHandler)))
	router.Get("/account", EnvHandler(repo, logger, AuthenticatedHandler(GetAccountHandler)))
	router.Patch("/account", EnvHandler(repo, logger, AuthenticatedHandler(UpdateAccountHandler)))

	return router
}

func getUserFromSession(req *http.Request, repo repository) *User {
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
	user, err := repo.GetUserBySessionID(sessionID)
	if err != nil {
		return nil
	}

	return user
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

	user := &User{}
	user.Name.SetCoerceZero(registration.Name, box.Empty)
	user.Email.SetCoerceZero(registration.Email, box.Empty)
	user.SetPassword(registration.Password)

	userID, err := env.repo.CreateUser(user)
	if err != nil {
		if err, ok := err.(DuplicationError); ok {
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

	err = env.repo.CreateSession(sessionID, userID)
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

	if err := env.repo.CreateSubscription(env.user.ID.MustGet(), subscription.URL); err != nil {
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

	if err := env.repo.DeleteSubscription(env.user.ID.MustGet(), int32(feedID)); err != nil {
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

	user, err := env.repo.GetUserByName(credentials.Name)
	if err != nil {
		w.WriteHeader(422)
		fmt.Fprintln(w, "Bad user name or password")
		return
	}

	if !user.IsPassword(credentials.Password) {
		w.WriteHeader(422)
		fmt.Fprintln(w, "Bad user name or password")
		return
	}

	sessionID, err := genSessionID()
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	err = env.repo.CreateSession(sessionID, user.ID.MustGet())
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
	err = env.repo.DeleteSession(sessionID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{Name: "sessionId", Value: "logged out", Expires: time.Unix(0, 0)}
	http.SetCookie(w, cookie)
}

func GetUnreadItemsHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	w.Header().Set("Content-Type", "application/json")
	if err := env.repo.CopyUnreadItemsAsJSONByUserID(w, env.user.ID.MustGet()); err != nil {
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

	err = env.repo.MarkItemRead(env.user.ID.MustGet(), int32(itemID))
	if err == notFound {
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
		err := env.repo.MarkItemRead(env.user.ID.MustGet(), itemID)
		if err != nil && err != notFound {
			http.Error(w, "Internal server error", http.StatusInternalServerError)
		}
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
			err := env.repo.CreateSubscription(env.user.ID.MustGet(), outline.URL)
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
	subs, err := env.repo.GetSubscriptions(env.user.ID.MustGet())
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}

	doc := OpmlDocument{Version: "1.0"}
	doc.Head.Title = "The Pithy Reader Export for " + env.user.Name.MustGet()

	for _, s := range subs {
		doc.Body.Outlines = append(doc.Body.Outlines, OpmlOutline{
			Text:  s.Name.MustGet(),
			Title: s.Name.MustGet(),
			Type:  "rss",
			URL:   s.URL.MustGet(),
		})
	}

	w.Header().Set("Content-Type", "application/xml")
	w.Header().Set("Content-Disposition", `attachment; filename="opml.xml"`)
	fmt.Fprint(w, xml.Header)
	xml.NewEncoder(w).Encode(doc)
}

func GetFeedsHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	w.Header().Set("Content-Type", "application/json")
	if err := env.repo.CopySubscriptionsForUserAsJSON(w, env.user.ID.MustGet()); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func GetAccountHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	var user struct {
		ID    box.Int32  `json:"id"`
		Name  box.String `json:"name"`
		Email box.String `json:"email"`
	}

	user.ID = env.user.ID
	user.Name = env.user.Name
	user.Email = env.user.Email

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

	if !env.user.IsPassword(update.ExistingPassword) {
		w.WriteHeader(422)
		fmt.Fprintln(w, "Bad existing password")
		return
	}

	user := &User{}
	user.Email.SetCoerceZero(update.Email, box.Empty)

	if update.NewPassword != "" {
		err := user.SetPassword(update.NewPassword)
		if err != nil {
			w.WriteHeader(422)
			fmt.Fprintln(w, err)
			return
		}
	}

	err := env.repo.UpdateUser(env.user.ID.MustGet(), user)
	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, `Internal server error`)
		env.logger.Error("UpdateUser", "err", err)
	}
}
