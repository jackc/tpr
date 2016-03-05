package main

import (
	"errors"
	"fmt"
	"github.com/jackc/cli"
	"github.com/jackc/pgx"
	"github.com/jackc/tpr/backend/data"
	"github.com/vaughan0/go-ini"
	log "gopkg.in/inconshreveable/log15.v2"
	"net/http"
	"net/http/httputil"
	"net/smtp"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
)

const version = "0.8.0"

type httpConfig struct {
	listenAddress string
	listenPort    string
	staticURL     string
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
				cli.StringFlag{"config, c", "tpr.conf", "path to config file"},
				cli.StringFlag{"static-url", "", "reverse proxy static asset requests to URL"},
			},
			Action: Serve,
		},
		{
			Name:        "reset-password",
			Usage:       "reset a user's password",
			Synopsis:    "[command options] username",
			Description: "reset a user's password",
			Flags: []cli.Flag{
				cli.StringFlag{"config, c", "tpr.conf", "path to config file"},
				cli.StringFlag{"password, p", "", "password to set"},
			},
			Action: ResetPassword,
		},
	}

	app.Run(os.Args)

}

func loadConfig(path string) (ini.File, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("Invalid config path: %v", err)
	}

	file, err := ini.LoadFile(path)
	if err != nil {
		return nil, fmt.Errorf("Failed to load config file: %v", err)
	}

	return file, nil
}

func newLogger(conf ini.File) (log.Logger, error) {
	level, _ := conf.Get("log", "level")
	if level == "" {
		level = "warn"
	}

	logger := log.New()
	setFilterHandler(level, logger, log.StdoutHandler)

	return logger, nil
}

func setFilterHandler(level string, logger log.Logger, handler log.Handler) error {
	if level == "none" {
		logger.SetHandler(log.DiscardHandler())
		return nil
	}

	lvl, err := log.LvlFromString(level)
	if err != nil {
		return fmt.Errorf("Bad log level: %v", err)
	}
	logger.SetHandler(log.LvlFilterHandler(lvl, handler))

	return nil
}

func newPool(conf ini.File, logger log.Logger) (*pgx.ConnPool, error) {
	logger = logger.New("module", "pgx")
	if level, ok := conf.Get("log", "pgx_level"); ok {
		setFilterHandler(level, logger, log.StdoutHandler)
	}

	connConfig := pgx.ConnConfig{Logger: logger}

	connConfig.Host, _ = conf.Get("database", "host")
	if connConfig.Host == "" {
		return nil, errors.New("Config must contain database.host but it does not")
	}

	if p, ok := conf.Get("database", "port"); ok {
		n, err := strconv.ParseUint(p, 10, 16)
		connConfig.Port = uint16(n)
		if err != nil {
			return nil, err
		}
	}

	var ok bool

	if connConfig.Database, ok = conf.Get("database", "database"); !ok {
		return nil, errors.New("Config must contain database.database but it does not")
	}
	connConfig.User, _ = conf.Get("database", "user")
	connConfig.Password, _ = conf.Get("database", "password")

	poolConfig := pgx.ConnPoolConfig{
		ConnConfig:     connConfig,
		MaxConnections: 10,
	}

	return pgx.NewConnPool(poolConfig)
}

func loadHTTPConfig(c *cli.Context, conf ini.File) (httpConfig, error) {
	config := httpConfig{}
	config.listenAddress = c.String("address")
	config.listenPort = c.String("port")
	config.staticURL = c.String("static-url")

	var ok bool
	if !c.IsSet("address") {
		if config.listenAddress, ok = conf.Get("server", "address"); !ok {
			return config, errors.New("Missing server address")
		}
	}

	if !c.IsSet("port") {
		if config.listenPort, ok = conf.Get("server", "port"); !ok {
			return config, errors.New("Missing server port")
		}
	}

	return config, nil
}

func newMailer(conf ini.File, logger log.Logger) (Mailer, error) {
	mailConf := conf.Section("mail")
	if len(mailConf) == 0 {
		return nil, nil
	}

	smtpAddr, ok := mailConf["smtp_server"]
	if !ok {
		return nil, errors.New("Missing mail -- smtp_server")
	}
	smtpPort, _ := mailConf["port"]
	if smtpPort == "" {
		smtpPort = "587"
	}

	fromAddr, ok := mailConf["from_address"]
	if !ok {
		return nil, errors.New("Missing mail -- from_address")
	}

	rootURL, ok := mailConf["root_url"]
	if !ok {
		return nil, errors.New("Missing mail -- root_url")
	}

	username, _ := mailConf["username"]
	password, _ := mailConf["password"]

	auth := smtp.PlainAuth("", username, password, smtpAddr)

	logger = logger.New("module", "mail")

	mailer := &SMTPMailer{
		ServerAddr: smtpAddr + ":" + smtpPort,
		Auth:       auth,
		From:       fromAddr,
		rootURL:    rootURL,
		logger:     logger,
	}

	return mailer, nil
}

func Serve(c *cli.Context) {
	conf, err := loadConfig(c.String("config"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	httpConfig, err := loadHTTPConfig(c, conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger, err := newLogger(conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	pool, err := newPool(conf, logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	mailer, err := newMailer(conf, logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	apiHandler := NewAPIHandler(pool, mailer, logger.New("module", "http"))
	http.Handle("/api/", http.StripPrefix("/api", apiHandler))

	if httpConfig.staticURL != "" {
		staticURL, err := url.Parse(httpConfig.staticURL)
		if err != nil {
			logger.Crit(fmt.Sprintf("Bad static-url: %v", err))
			os.Exit(1)
		}
		http.Handle("/", httputil.NewSingleHostReverseProxy(staticURL))
	}

	listenAt := fmt.Sprintf("%s:%s", httpConfig.listenAddress, httpConfig.listenPort)
	fmt.Printf("Starting to listen on: %s\n", listenAt)

	feedUpdater := NewFeedUpdater(pool, logger.New("module", "feedUpdater"))
	go feedUpdater.KeepFeedsFresh()

	if err := http.ListenAndServe(listenAt, nil); err != nil {
		os.Stderr.WriteString("Could not start web server!\n")
		os.Exit(1)
	}
}

func ResetPassword(c *cli.Context) {
	if len(c.Args()) != 1 {
		cli.ShowCommandHelp(c, c.Command.Name)
		os.Exit(1)
	}

	name := c.Args()[0]

	conf, err := loadConfig(c.String("config"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	logger, err := newLogger(conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	pool, err := newPool(conf, logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	user, err := data.SelectUserByName(pool, name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	password, err := genRandPassword()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	update := &data.User{}
	SetPassword(update, password)

	err = data.UpdateUser(pool, user.ID.Value, update)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("User:", name)
	fmt.Println("Password:", password)
}
