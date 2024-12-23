package main

import (
	"context"
	"errors"
	"fmt"
	"net/smtp"
	"os"
	"path/filepath"
	"strconv"

	log15adapter "github.com/jackc/pgx-log15"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/jackc/tpr/backend"
	"github.com/jackc/tpr/backend/data"
	"github.com/urfave/cli"
	"github.com/vaughan0/go-ini"
	log "gopkg.in/inconshreveable/log15.v2"
)

const version = "0.8.1"

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
			Description: "run the tpr server",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "address, a", Value: "127.0.0.1", Usage: "address to listen on"},
				cli.StringFlag{Name: "port, p", Value: "8080", Usage: "port to listen on"},
				cli.StringFlag{Name: "config, c", Value: "tpr.conf", Usage: "path to config file"},
				cli.StringFlag{Name: "static-url", Value: "", Usage: "reverse proxy static asset requests to URL"},
			},
			Action: Serve,
		},
		{
			Name:        "reset-password",
			Usage:       "reset a user's password",
			Description: "reset a user's password",
			Flags: []cli.Flag{
				cli.StringFlag{Name: "config, c", Value: "tpr.conf", Usage: "path to config file"},
				cli.StringFlag{Name: "password, p", Value: "", Usage: "password to set"},
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

func newPool(conf ini.File, logger log.Logger) (*pgxpool.Pool, error) {
	logger = logger.New("module", "pgx")
	if level, ok := conf.Get("log", "pgx_level"); ok {
		setFilterHandler(level, logger, log.StdoutHandler)
	}

	config, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, err
	}
	config.ConnConfig.Tracer = &tracelog.TraceLog{Logger: log15adapter.NewLogger(logger), LogLevel: tracelog.LogLevelInfo}

	config.ConnConfig.Host, _ = conf.Get("database", "host")
	if config.ConnConfig.Host == "" {
		return nil, errors.New("Config must contain database.host but it does not")
	}

	if p, ok := conf.Get("database", "port"); ok {
		n, err := strconv.ParseUint(p, 10, 16)
		config.ConnConfig.Port = uint16(n)
		if err != nil {
			return nil, err
		}
	}

	var ok bool

	if config.ConnConfig.Database, ok = conf.Get("database", "database"); !ok {
		return nil, errors.New("Config must contain database.database but it does not")
	}
	config.ConnConfig.User, _ = conf.Get("database", "user")
	config.ConnConfig.Password, _ = conf.Get("database", "password")

	config.MaxConns = 10

	return pgxpool.NewWithConfig(context.Background(), config)
}

func loadHTTPConfig(c *cli.Context, conf ini.File) (backend.HTTPConfig, error) {
	config := backend.HTTPConfig{}
	config.ListenAddress = c.String("address")
	config.ListenPort = c.String("port")
	config.StaticURL = c.String("static-url")

	var ok bool
	if !c.IsSet("address") {
		if config.ListenAddress, ok = conf.Get("server", "address"); !ok {
			return config, errors.New("Missing server address")
		}
	}

	if !c.IsSet("port") {
		if config.ListenPort, ok = conf.Get("server", "port"); !ok {
			return config, errors.New("Missing server port")
		}
	}

	return config, nil
}

func newMailer(conf ini.File, logger log.Logger) (backend.Mailer, error) {
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

	mailer := &backend.SMTPMailer{
		ServerAddr: smtpAddr + ":" + smtpPort,
		Auth:       auth,
		From:       fromAddr,
		RootURL:    rootURL,
		Logger:     logger,
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

	feedUpdater := backend.NewFeedUpdater(pool, logger.New("module", "feedUpdater"))
	go feedUpdater.KeepFeedsFresh()

	server, err := backend.NewAppServer(httpConfig, pool, mailer, logger)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create web server: %v\n", err)
		os.Exit(1)
	}

	err = server.Serve()
	if err != nil {
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

	user, err := data.SelectUserByName(context.Background(), pool, name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	password, err := backend.GenRandPassword()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	update := &data.User{Name: pgtype.Text{String: name, Valid: true}}
	backend.SetPassword(update, password)

	err = data.UpdateUser(context.Background(), pool, user.ID.Int32, update)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fmt.Println("User:", name)
	fmt.Println("Password:", password)
}
