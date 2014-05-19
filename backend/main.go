package main

import (
	"errors"
	"fmt"
	"github.com/JackC/cli"
	"github.com/JackC/pgx"
	qv "github.com/JackC/quo_vadis"
	"github.com/vaughan0/go-ini"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

const version = "0.5.1"

var repo repository

var config struct {
	configPath    string
	listenAddress string
	listenPort    string
	staticURL     string
}

func extractConnConfig(file ini.File) (connConfig pgx.ConnConfig, err error) {
	connConfig.Host, _ = file.Get("database", "host")
	connConfig.Socket, _ = file.Get("database", "socket")
	if connConfig.Host == "" && connConfig.Socket == "" {
		err = errors.New("Config must contain database.host or database.socket but it does not")
		return
	}

	if p, ok := file.Get("database", "port"); ok {
		n, err := strconv.ParseUint(p, 10, 16)
		connConfig.Port = uint16(n)
		if err != nil {
			return connConfig, err
		}
	}

	var ok bool

	if connConfig.Database, ok = file.Get("database", "database"); !ok {
		err = errors.New("Config must contain database.database but it does not")
		return
	}
	if connConfig.User, ok = file.Get("database", "user"); !ok {
		err = errors.New("Config must contain database.user but it does not")
		return
	}
	connConfig.Password, _ = file.Get("database", "password")
	return
}

func main() {
	app := cli.NewApp()
	app.Name = "tpr"
	app.Usage = "The Pithy Reader RSS Aggregator"
	app.Version = version
	app.Author = "Jack Christensen"
	app.Email = "jack@jackchristensen.com"

	app.Commands = []cli.Command{
		{
			Name:        "server",
			ShortName:   "s",
			Usage:       "run the server",
			Synopsis:    "[command options]",
			Description: "run the tpr server",
			Flags: []cli.Flag{
				cli.StringFlag{"address, a", "127.0.0.1", "address to listen on"},
				cli.StringFlag{"port, p", "8080", "port to listen on"},
				cli.StringFlag{"config, c", "config.conf", "path to config file"},
				cli.StringFlag{"static-url", "", "reverse proxy static asset requests to URL"},
			},
			Action: Serve,
		},
	}

	app.Run(os.Args)

}

func Serve(c *cli.Context) {
	var err error

	config.listenAddress = c.String("address")
	config.listenPort = c.String("port")
	config.configPath = c.String("config")
	config.staticURL = c.String("static-url")

	if config.configPath, err = filepath.Abs(config.configPath); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid config path: %v\n", err)
		os.Exit(1)
	}

	file, err := ini.LoadFile(config.configPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	var ok bool

	if !c.IsSet("address") {
		if config.listenAddress, ok = file.Get("server", "address"); !ok {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}

	if !c.IsSet("port") {
		if config.listenPort, ok = file.Get("server", "port"); !ok {
			fmt.Fprintln(os.Stderr, err.Error())
			os.Exit(1)
		}
	}

	poolConfig := pgx.ConnPoolConfig{MaxConnections: 10, AfterConnect: afterConnect}
	if poolConfig.ConnConfig, err = extractConnConfig(file); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	poolConfig.Logger = &PackageLogger{logger: logger, pkg: "pgx"}

	repo, err = NewPgxRepository(poolConfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to create pgx repository: %v\n", err)
		os.Exit(1)
	}

	router := qv.NewRouter()

	router.Post("/register", http.HandlerFunc(RegisterHandler))
	router.Post("/sessions", http.HandlerFunc(CreateSessionHandler))
	router.Delete("/sessions/:id", http.HandlerFunc(DeleteSessionHandler))
	router.Post("/subscriptions", ApiSecureHandlerFunc(CreateSubscriptionHandler))
	router.Delete("/subscriptions/:id", ApiSecureHandlerFunc(DeleteSubscriptionHandler))
	router.Get("/feeds", ApiSecureHandlerFunc(GetFeedsHandler))
	router.Post("/feeds/import", ApiSecureHandlerFunc(ImportFeedsHandler))
	router.Get("/items/unread", ApiSecureHandlerFunc(GetUnreadItemsHandler))
	router.Post("/items/unread/mark_multiple_read", ApiSecureHandlerFunc(MarkMultipleItemsReadHandler))
	router.Delete("/items/unread/:id", ApiSecureHandlerFunc(MarkItemReadHandler))
	http.Handle("/api/", http.StripPrefix("/api", router))

	if config.staticURL != "" {
		staticURL, err := url.Parse(config.staticURL)
		if err != nil {
			logger.Fatal("tpr", fmt.Sprintf("Bad static-url: %v", err))
			os.Exit(1)
		}
		http.Handle("/", httputil.NewSingleHostReverseProxy(staticURL))
	}

	listenAt := fmt.Sprintf("%s:%s", config.listenAddress, config.listenPort)
	fmt.Printf("Starting to listen on: %s\n", listenAt)

	go KeepFeedsFresh()

	if err := http.ListenAndServe(listenAt, nil); err != nil {
		os.Stderr.WriteString("Could not start web server!\n")
		os.Exit(1)
	}
}
