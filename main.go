package main

import (
	"bytes"
	"code.google.com/p/go.crypto/scrypt"
	"errors"
	"flag"
	"fmt"
	"github.com/JackC/form"
	"github.com/JackC/pgx"
	qv "github.com/JackC/quo_vadis"
	"github.com/kylelemons/go-gypsy/yaml"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var pool *pgx.ConnectionPool

var config struct {
	configPath    string
	listenAddress string
	listenPort    string
}

var loginFormTemplate *form.FormTemplate
var registrationFormTemplate *form.FormTemplate
var subscriptionFormTemplate *form.FormTemplate

func init() {
	var err error
	var yf *yaml.File

	flag.StringVar(&config.listenAddress, "address", "127.0.0.1", "address to listen on")
	flag.StringVar(&config.listenPort, "port", "8080", "port to listen on")
	flag.StringVar(&config.configPath, "config", "config.yml", "path to config file")
	flag.Parse()

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

	if err = migrate(connectionParameters); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	poolOptions := pgx.ConnectionPoolOptions{MaxConnections: 5, AfterConnect: afterConnect}
	pool, err = pgx.NewConnectionPool(connectionParameters, poolOptions)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create database connection pool: %v\n", err)
		os.Exit(1)
	}

	registrationFormTemplate = form.NewFormTemplate()
	registrationFormTemplate.AddField(&form.StringTemplate{Name: "name", Required: true, MaxLength: 30})
	registrationFormTemplate.AddField(&form.StringTemplate{Name: "password", Required: true, MinLength: 8, MaxLength: 50})
	registrationFormTemplate.AddField(&form.StringTemplate{Name: "passwordConfirmation", Required: true, MaxLength: 50})
	registrationFormTemplate.CustomValidate = func(f *form.Form) {
		password := f.Fields["password"]
		confirmation := f.Fields["passwordConfirmation"]
		if password.Error == nil && confirmation.Error == nil && password.Parsed != confirmation.Parsed {
			confirmation.Error = errors.New("does not match password")
		}
	}

	subscriptionFormTemplate = form.NewFormTemplate()
	subscriptionFormTemplate.AddField(&form.StringTemplate{Name: "url", Required: true, MaxLength: 8192})

	loginFormTemplate = form.NewFormTemplate()
	loginFormTemplate.AddField(&form.StringTemplate{Name: "name", Required: true, MaxLength: 30})
	loginFormTemplate.AddField(&form.StringTemplate{Name: "password", Required: true, MaxLength: 50})
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

// afterConnect creates the prepared statements that this application uses
func afterConnect(conn *pgx.Connection) (err error) {
	return
}

type SecureHandlerFunc func(w http.ResponseWriter, req *http.Request, env *environment)

func (f SecureHandlerFunc) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	env := CreateEnvironment(req)
	if env.CurrentAccount() == nil {
		http.Redirect(w, req, "/login", http.StatusSeeOther)
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
		var cookie *http.Cookie
		var session Session
		var err error
		var present bool

		if cookie, err = env.request.Cookie("sessionId"); err != nil {
			return nil
		}

		if session, present = getSession(cookie.Value); !present {
			return nil
		}

		var name interface{}
		// TODO - this could be an error from no records found -- or the connection could be dead or we could have a syntax error...
		name, err = pool.SelectValue("select name from users where id=$1", session.userID)
		if err == nil {
			env.currentAccount = &currentAccount{id: session.userID, name: name.(string)}
		}
	}
	return env.currentAccount
}

func RegistrationFormHandler(w http.ResponseWriter, req *http.Request) {
	RenderRegister(w, registrationFormTemplate.New())
}

func RegisterHandler(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	f := registrationFormTemplate.Parse(req.Form)
	registrationFormTemplate.Validate(f)
	if f.IsValid() {
		if userID, err := CreateUser(f.Fields["name"].Parsed.(string), f.Fields["password"].Parsed.(string)); err == nil {
			sessionId := createSession(userID)
			cookie := createSessionCookie(sessionId)
			http.SetCookie(w, cookie)
			http.Redirect(w, req, "/subscribe", http.StatusSeeOther)
		} else {
			if strings.Contains(err.Error(), "users_name_unq") {
				f.Fields["name"].Error = errors.New("User name is already taken")
			} else {
				panic(err.Error())
			}
			RenderRegister(w, f)
		}
	} else {
		RenderRegister(w, f)
	}
}

func SubscriptionFormHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	RenderSubscribe(w, env, subscriptionFormTemplate.New())
}

func SubscribeHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	req.ParseForm()
	f := subscriptionFormTemplate.Parse(req.Form)
	subscriptionFormTemplate.Validate(f)
	if f.IsValid() {
		if err := Subscribe(env.CurrentAccount().id, f.Fields["url"].Parsed.(string)); err == nil {
			http.Redirect(w, req, "/feeds", http.StatusSeeOther)
		} else {
			panic(err.Error())
		}
	} else {
		RenderSubscribe(w, env, f)
	}
}

func AuthenticateUser(name, password string) (userID int32, err error) {
	var passwordDigest []byte
	var passwordSalt []byte

	err = pool.SelectFunc("select id, password_digest, password_salt from users where name=$1", func(r *pgx.DataRowReader) (err error) {
		userID = r.ReadValue().(int32)
		passwordDigest = r.ReadValue().([]byte)
		passwordSalt = r.ReadValue().([]byte)
		return
	}, name)

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

func LoginFormHandler(w http.ResponseWriter, req *http.Request) {
	RenderLogin(w, loginFormTemplate.New())
}

func LoginHandler(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	f := loginFormTemplate.Parse(req.Form)
	loginFormTemplate.Validate(f)

	if !f.IsValid() {
		RenderLogin(w, f)
		return
	}

	if userID, err := AuthenticateUser(f.Fields["name"].Parsed.(string), f.Fields["password"].Parsed.(string)); err == nil {
		sessionId := createSession(userID)
		cookie := createSessionCookie(sessionId)
		http.SetCookie(w, cookie)
		http.Redirect(w, req, "/feeds", http.StatusSeeOther)
	} else {
		RenderLogin(w, f)
	}
}

func createSessionCookie(sessionId string) *http.Cookie {
	return &http.Cookie{Name: "sessionId", Value: sessionId}
}

func FeedsIndexHandler(w http.ResponseWriter, req *http.Request, env *environment) {
	feeds, err := GetFeedsForUserID(env.CurrentAccount().id)
	if err != nil {
		panic("unable to find feeds")
	}

	RenderFeedsIndex(w, env, feeds)
}

func main() {
	router := qv.NewRouter()
	router.Get("/login", http.HandlerFunc(LoginFormHandler))
	router.Post("/login", http.HandlerFunc(LoginHandler))
	router.Get("/subscribe", SecureHandlerFunc(SubscriptionFormHandler))
	router.Post("/subscribe", SecureHandlerFunc(SubscribeHandler))
	router.Get("/register", http.HandlerFunc(RegistrationFormHandler))
	router.Post("/register", http.HandlerFunc(RegisterHandler))
	router.Get("/feeds", SecureHandlerFunc(FeedsIndexHandler))
	http.Handle("/", router)

	listenAt := fmt.Sprintf("%s:%s", config.listenAddress, config.listenPort)
	fmt.Printf("Starting to listen on: %s\n", listenAt)

	if err := http.ListenAndServe(listenAt, nil); err != nil {
		os.Stderr.WriteString("Could not start web server!\n")
		os.Exit(1)
	}
}
