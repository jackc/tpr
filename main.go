package main

import (
	"bytes"
	"code.google.com/p/go.crypto/scrypt"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"flag"
	"fmt"
	"github.com/JackC/pgx"
	qv "github.com/JackC/quo_vadis"
	"github.com/kylelemons/go-gypsy/yaml"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const version = "0.1.0"

var repo repository

var config struct {
	configPath    string
	listenAddress string
	listenPort    string
}

func initialize() {
	var err error
	var yf *yaml.File

	flag.StringVar(&config.listenAddress, "address", "127.0.0.1", "address to listen on")
	flag.StringVar(&config.listenPort, "port", "8080", "port to listen on")
	flag.StringVar(&config.configPath, "config", "config.yml", "path to config file")

	var printVersion bool
	flag.BoolVar(&printVersion, "version", false, "Print version and exit")
	flag.Parse()

	if printVersion {
		fmt.Printf("tpr v%s\n", version)
		os.Exit(0)
	}

	givenCliArgs := make(map[string]bool)
	flag.Visit(func(f *flag.Flag) {
		givenCliArgs[f.Name] = true
	})

	if config.configPath, err = filepath.Abs(config.configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid config path: %v\n", err)
		os.Exit(1)
	}

	if yf, err = yaml.ReadFile(config.configPath); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	if !givenCliArgs["address"] {
		if address, err := yf.Get("address"); err == nil {
			config.listenAddress = address
		}
	}

	if !givenCliArgs["port"] {
		if port, err := yf.Get("port"); err == nil {
			config.listenPort = port
		}
	}

	var connectionParameters pgx.ConnectionParameters
	if connectionParameters, err = extractConnectionOptions(yf); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	pgxLogger := &PackageLogger{logger: logger, pkg: "pgx"}
	connectionParameters.Logger = pgxLogger

	if err = migrate(connectionParameters); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	poolOptions := pgx.ConnectionPoolOptions{MaxConnections: 10, AfterConnect: afterConnect, Logger: pgxLogger}

	repo, err = NewPgxRepository(connectionParameters, poolOptions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create pgx repository: %v\n", err)
		os.Exit(1)
	}
}

func extractConnectionOptions(config *yaml.File) (connectionOptions pgx.ConnectionParameters, err error) {
	connectionOptions.Host, _ = config.Get("database.host")
	connectionOptions.Socket, _ = config.Get("database.socket")
	if connectionOptions.Host == "" && connectionOptions.Socket == "" {
		err = errors.New("Config must contain database.host or database.socket but it does not")
		return
	}
	port, _ := config.GetInt("database.port")
	connectionOptions.Port = uint16(port)
	if connectionOptions.Database, err = config.Get("database.database"); err != nil {
		err = errors.New("Config must contain database.database but it does not")
		return
	}
	if connectionOptions.User, err = config.Get("database.user"); err != nil {
		err = errors.New("Config must contain database.user but it does not")
		return
	}
	connectionOptions.Password, _ = config.Get("database.password")
	return
}

type ApiSecureHandlerFunc func(w http.ResponseWriter, req *http.Request, env *environment)

func (f ApiSecureHandlerFunc) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	env := CreateEnvironment(req)
	if env.CurrentAccount() == nil {
		w.WriteHeader(http.StatusForbidden)
		fmt.Fprint(w, "Bad or missing sessionID")
		return
	}
	f(w, req, env)
}

type currentAccount struct {
	id   int32
	name string
}

type environment struct {
	request        *http.Request
	currentAccount *currentAccount
}

func CreateEnvironment(req *http.Request) *environment {
	return &environment{request: req}
}

func (env *environment) CurrentAccount() *currentAccount {
	if env.currentAccount == nil {
		var session Session
		var err error
		var present bool

		var sessionID []byte
		sessionID, err = hex.DecodeString(env.request.FormValue("sessionID"))
		if err != nil {
			logger.Warning("tpr", fmt.Sprintf(`Bad or missing to sessionID "%s": %v`, env.request.FormValue("sessionID"), err))
			return nil
		}
		if session, present = getSession(sessionID); !present {
			return nil
		}

		var name interface{}
		// TODO - this could be an error from no records found -- or the connection could be dead or we could have a syntax error...
		name, err = repo.getUserName(session.userID)
		if err == nil {
			env.currentAccount = &currentAccount{id: session.userID, name: name.(string)}
		}
	}
	return env.currentAccount
}

func RegisterHandler(w http.ResponseWriter, req *http.Request) {
	var registration struct {
		Name                 string `json:"name"`
		Password             string `json:"password"`
		PasswordConfirmation string `json:"passwordConfirmation"`
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

	if len(registration.Password) < 8 {
		w.WriteHeader(422)
		fmt.Fprintln(w, `"password" must be at least than 8 characters`)
		return
	}

	if registration.Password != registration.PasswordConfirmation {
		w.WriteHeader(422)
		fmt.Fprintln(w, `"passwordConfirmation" must equal "password"`)
		return
	}

	if userID, err := CreateUser(registration.Name, registration.Password); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		var response struct {
			Name      string `json:"name"`
			SessionID string `json:"sessionID"`
		}

		response.Name = registration.Name
		response.SessionID = hex.EncodeToString(createSession(userID))

		encoder := json.NewEncoder(w)
		encoder.Encode(response)
	} else {
		if strings.Contains(err.Error(), "users_name_unq") {
			w.WriteHeader(422)
			fmt.Fprintln(w, `"name" is already taken`)
			return
		} else {
			panic(err.Error())
		}
	}
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

	if err := Subscribe(env.CurrentAccount().id, subscription.URL); err != nil {
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

	if err := repo.deleteSubscription(env.CurrentAccount().id, int32(feedID)); err != nil {
		w.WriteHeader(422)
		fmt.Fprintf(w, "Error deleting subscription: %v", err)
		return
	}

	w.WriteHeader(http.StatusOK)
}

func AuthenticateUser(name, password string) (userID int32, err error) {
	userID, passwordDigest, passwordSalt, err := repo.getUserAuthenticationByName(name)
	if err != nil {
		return
	}

	var digest []byte
	digest, _ = scrypt.Key([]byte(password), passwordSalt, 16384, 8, 1, 32)

	if !bytes.Equal(digest, passwordDigest) {
		err = fmt.Errorf("Bad user name or password")
	}
	return
}

func CreateSessionHandler(w http.ResponseWriter, req *http.Request) {
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

	if userID, err := AuthenticateUser(credentials.Name, credentials.Password); err == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)

		var response struct {
			Name      string `json:"name"`
			SessionID string `json:"sessionID"`
		}

		response.Name = credentials.Name
		response.SessionID = hex.EncodeToString(createSession(userID))

		encoder := json.NewEncoder(w)
		encoder.Encode(response)
	} else {
		w.WriteHeader(422)
		fmt.Fprintln(w, `Bad user name or password`)
		return
	}
}

func DeleteSessionHandler(w http.ResponseWriter, req *http.Request) {
	sessionID, err := hex.DecodeString(req.FormValue("id"))
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	err = deleteSession(sessionID)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	cookie := &http.Cookie{Name: "sessionId", Value: "logged out", Expires: time.Unix(0, 0)}
	http.SetCookie(w, cookie)
}

func GetUnreadItemsHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	w.Header().Set("Content-Type", "application/json")
	if err := repo.copyUnreadItemsAsJSONByUserID(w, env.CurrentAccount().id); err != nil {
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

	err = repo.markItemRead(env.CurrentAccount().id, int32(itemID))
	if err == notFound {
		http.NotFound(w, req)
		return
	}
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func MarkAllItemsReadHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	err := repo.markAllItemsRead(env.CurrentAccount().id)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func createSessionCookie(sessionId []byte) *http.Cookie {
	return &http.Cookie{Name: "sessionId", Value: hex.EncodeToString(sessionId)}
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
			err := Subscribe(env.CurrentAccount().id, outline.URL)
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

func GetFeedsHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	w.Header().Set("Content-Type", "application/json")
	if err := repo.copyFeedsAsJSONBySubscribedUserID(w, env.CurrentAccount().id); err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func NoDirListing(handler http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/") {
			http.NotFound(w, r)
			return
		}
		handler.ServeHTTP(w, r)
	})
}

func PreGzipped(handler http.Handler) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if strings.Contains(req.Header.Get("Accept-Encoding"), "gzip") {
			contentType := mime.TypeByExtension(filepath.Ext(req.URL.Path))
			if contentType != "" {
				w.Header().Set("Content-Type", contentType)
			}

			req.URL.Path = req.URL.Path + ".gz"
			w.Header().Set("Content-Encoding", "gzip")
		}
		handler.ServeHTTP(w, req)
	})
}

func IndexHtmlHandler(w http.ResponseWriter, req *http.Request) {
	http.ServeFile(w, req, "public/index.html")
}

func AssetFileServer(root http.FileSystem) http.HandlerFunc {
	return NoDirListing(PreGzipped(http.FileServer(root)))
}

func main() {
	initialize()
	router := qv.NewRouter()

	router.Post("/register", http.HandlerFunc(RegisterHandler))
	router.Post("/sessions", http.HandlerFunc(CreateSessionHandler))
	router.Delete("/sessions/:id", http.HandlerFunc(DeleteSessionHandler))
	router.Post("/subscriptions", ApiSecureHandlerFunc(CreateSubscriptionHandler))
	router.Delete("/subscriptions/:id", ApiSecureHandlerFunc(DeleteSubscriptionHandler))
	router.Get("/feeds", ApiSecureHandlerFunc(GetFeedsHandler))
	router.Post("/feeds/import", ApiSecureHandlerFunc(ImportFeedsHandler))
	router.Get("/items/unread", ApiSecureHandlerFunc(GetUnreadItemsHandler))
	router.Delete("/items/unread", ApiSecureHandlerFunc(MarkAllItemsReadHandler))
	router.Delete("/items/unread/:id", ApiSecureHandlerFunc(MarkItemReadHandler))
	http.Handle("/api/", http.StripPrefix("/api", router))

	http.Handle("/", http.HandlerFunc(IndexHtmlHandler))
	http.Handle("/css/", AssetFileServer(http.Dir("./public/")))
	http.Handle("/js/", AssetFileServer(http.Dir("./public/")))

	listenAt := fmt.Sprintf("%s:%s", config.listenAddress, config.listenPort)
	fmt.Printf("Starting to listen on: %s\n", listenAt)

	go KeepFeedsFresh()

	if err := http.ListenAndServe(listenAt, nil); err != nil {
		os.Stderr.WriteString("Could not start web server!\n")
		os.Exit(1)
	}
}
