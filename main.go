package main

import (
	"context"
	"errors"
	"fmt"
	"net/smtp"
	"os"
	"strconv"

	log15adapter "github.com/jackc/pgx-log15"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/jackc/tpr/backend"
	"github.com/jackc/tpr/backend/data"
	"github.com/urfave/cli"
	log "gopkg.in/inconshreveable/log15.v2"
)

const version = "0.8.1"

// Shared database flags for both commands
var databaseFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "db-host",
		EnvVar: "TPR_DB_HOST",
		Usage:  "PostgreSQL host",
	},
	cli.StringFlag{
		Name:   "db-port",
		Value:  "5432",
		EnvVar: "TPR_DB_PORT",
		Usage:  "PostgreSQL port",
	},
	cli.StringFlag{
		Name:   "db-name",
		EnvVar: "TPR_DB_NAME",
		Usage:  "PostgreSQL database name",
	},
	cli.StringFlag{
		Name:   "db-user",
		EnvVar: "TPR_DB_USER",
		Usage:  "PostgreSQL user",
	},
	cli.StringFlag{
		Name:   "db-password",
		EnvVar: "TPR_DB_PASSWORD",
		Usage:  "PostgreSQL password",
	},
}

// Shared log flags for both commands
var logFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "log-level",
		Value:  "warn",
		EnvVar: "TPR_LOG_LEVEL",
		Usage:  "Log level (debug, info, warn, error, crit, none)",
	},
	cli.StringFlag{
		Name:   "pgx-log-level",
		Value:  "warn",
		EnvVar: "TPR_PGX_LOG_LEVEL",
		Usage:  "PGX (database driver) log level",
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "tpr"
	app.Usage = "The Pithy Reader RSS Aggregator"
	app.Version = version
	app.Author = "Jack Christensen"
	app.Email = "jack@jackchristensen.com"

	serverFlags := []cli.Flag{
		cli.StringFlag{
			Name:   "address, a",
			Value:  "127.0.0.1",
			EnvVar: "TPR_ADDRESS",
			Usage:  "address to listen on",
		},
		cli.StringFlag{
			Name:   "port, p",
			Value:  "8080",
			EnvVar: "TPR_PORT",
			Usage:  "port to listen on",
		},
		cli.StringFlag{
			Name:   "static-url",
			Value:  "",
			EnvVar: "TPR_STATIC_URL",
			Usage:  "reverse proxy static asset requests to URL",
		},
		cli.StringFlag{
			Name:   "smtp-server",
			EnvVar: "TPR_SMTP_SERVER",
			Usage:  "SMTP server hostname",
		},
		cli.StringFlag{
			Name:   "smtp-port",
			Value:  "587",
			EnvVar: "TPR_SMTP_PORT",
			Usage:  "SMTP server port",
		},
		cli.StringFlag{
			Name:   "smtp-username",
			EnvVar: "TPR_SMTP_USERNAME",
			Usage:  "SMTP username",
		},
		cli.StringFlag{
			Name:   "smtp-password",
			EnvVar: "TPR_SMTP_PASSWORD",
			Usage:  "SMTP password",
		},
		cli.StringFlag{
			Name:   "smtp-from",
			EnvVar: "TPR_SMTP_FROM",
			Usage:  "From address for emails",
		},
		cli.StringFlag{
			Name:   "root-url",
			EnvVar: "TPR_ROOT_URL",
			Usage:  "Root URL for email links (e.g., https://example.com)",
		},
	}
	serverFlags = append(serverFlags, databaseFlags...)
	serverFlags = append(serverFlags, logFlags...)

	resetPasswordFlags := []cli.Flag{
		cli.StringFlag{
			Name:  "password, p",
			Value: "",
			Usage: "password to set",
		},
	}
	resetPasswordFlags = append(resetPasswordFlags, databaseFlags...)
	resetPasswordFlags = append(resetPasswordFlags, logFlags...)

	app.Commands = []cli.Command{
		{
			Name:        "server",
			ShortName:   "s",
			Usage:       "run the server",
			Description: "run the tpr server",
			Flags:       serverFlags,
			Action:      Serve,
		},
		{
			Name:        "reset-password",
			Usage:       "reset a user's password",
			Description: "reset a user's password",
			Flags:       resetPasswordFlags,
			Action:      ResetPassword,
		},
	}

	app.Run(os.Args)
}

func newLogger(c *cli.Context) (log.Logger, error) {
	level := c.String("log-level")
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

func newPool(c *cli.Context, logger log.Logger) (*pgxpool.Pool, error) {
	logger = logger.New("module", "pgx")
	if level := c.String("pgx-log-level"); level != "" {
		setFilterHandler(level, logger, log.StdoutHandler)
	}

	config, err := pgxpool.ParseConfig("")
	if err != nil {
		return nil, err
	}
	config.ConnConfig.Tracer = &tracelog.TraceLog{Logger: log15adapter.NewLogger(logger), LogLevel: tracelog.LogLevelInfo}

	config.ConnConfig.Host = c.String("db-host")
	if config.ConnConfig.Host == "" {
		return nil, errors.New("--db-host is required")
	}

	if p := c.String("db-port"); p != "" {
		n, err := strconv.ParseUint(p, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("invalid db-port: %v", err)
		}
		config.ConnConfig.Port = uint16(n)
	}

	config.ConnConfig.Database = c.String("db-name")
	if config.ConnConfig.Database == "" {
		return nil, errors.New("--db-name is required")
	}

	config.ConnConfig.User = c.String("db-user")
	config.ConnConfig.Password = c.String("db-password")

	config.MaxConns = 10

	return pgxpool.NewWithConfig(context.Background(), config)
}

func newMailer(c *cli.Context, logger log.Logger) (backend.Mailer, error) {
	smtpServer := c.String("smtp-server")
	if smtpServer == "" {
		// Mail is optional - return nil if not configured
		return nil, nil
	}

	smtpPort := c.String("smtp-port")
	if smtpPort == "" {
		smtpPort = "587"
	}

	fromAddr := c.String("smtp-from")
	if fromAddr == "" {
		return nil, errors.New("--smtp-from is required when --smtp-server is set")
	}

	rootURL := c.String("root-url")
	if rootURL == "" {
		return nil, errors.New("--root-url is required when --smtp-server is set")
	}

	username := c.String("smtp-username")
	password := c.String("smtp-password")

	auth := smtp.PlainAuth("", username, password, smtpServer)

	logger = logger.New("module", "mail")

	mailer := &backend.SMTPMailer{
		ServerAddr: smtpServer + ":" + smtpPort,
		Auth:       auth,
		From:       fromAddr,
		RootURL:    rootURL,
		Logger:     logger,
	}

	return mailer, nil
}

func Serve(c *cli.Context) {
	httpConfig := backend.HTTPConfig{
		ListenAddress: c.String("address"),
		ListenPort:    c.String("port"),
		StaticURL:     c.String("static-url"),
	}

	logger, err := newLogger(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	pool, err := newPool(c, logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	mailer, err := newMailer(c, logger)
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

	logger, err := newLogger(c)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	pool, err := newPool(c, logger)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	user, err := data.SelectUserByName(context.Background(), pool, name)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	password := c.String("password")
	if password == "" {
		password, err = backend.GenRandPassword()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
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
