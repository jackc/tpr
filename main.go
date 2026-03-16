package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/smtp"
	"os"

	pgxslog "github.com/mcosta74/pgx-slog"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/jackc/tpr/backend"
	"github.com/jackc/tpr/backend/data"
	"github.com/urfave/cli"
)

const version = "0.8.1"

// Shared database flags for both commands
var databaseFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "database-url",
		EnvVar: "DATABASE_URL",
		Usage:  "PostgreSQL connection URL (e.g. postgres://user:pass@host:port/dbname)",
	},
}

// Shared log flags for both commands
var logFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "log-level",
		Value:  "warn",
		EnvVar: "LOG_LEVEL",
		Usage:  "Log level (debug, info, warn, error, crit, none)",
	},
	cli.StringFlag{
		Name:   "log-format",
		Value:  "text",
		EnvVar: "LOG_FORMAT",
		Usage:  "Log format (text, json, journal)",
	},
	cli.StringFlag{
		Name:   "pgx-log-level",
		Value:  "warn",
		EnvVar: "PGX_LOG_LEVEL",
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
			Name:   "bind-address, a",
			Value:  "127.0.0.1",
			EnvVar: "BIND_ADDRESS",
			Usage:  "address to listen on",
		},
		cli.StringFlag{
			Name:   "port, p",
			Value:  "8080",
			EnvVar: "PORT",
			Usage:  "port to listen on",
		},
		cli.StringFlag{
			Name:   "smtp-server",
			EnvVar: "SMTP_SERVER",
			Usage:  "SMTP server hostname",
		},
		cli.StringFlag{
			Name:   "smtp-port",
			Value:  "587",
			EnvVar: "SMTP_PORT",
			Usage:  "SMTP server port",
		},
		cli.StringFlag{
			Name:   "smtp-username",
			EnvVar: "SMTP_USERNAME",
			Usage:  "SMTP username",
		},
		cli.StringFlag{
			Name:   "smtp-password",
			EnvVar: "SMTP_PASSWORD",
			Usage:  "SMTP password",
		},
		cli.StringFlag{
			Name:   "smtp-from",
			EnvVar: "SMTP_FROM",
			Usage:  "From address for emails",
		},
		cli.StringFlag{
			Name:   "root-url",
			EnvVar: "ROOT_URL",
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

func newLogger(c *cli.Context) (*slog.Logger, error) {
	level := c.String("log-level")
	if level == "" {
		level = "warn"
	}
	format := c.String("log-format")
	if format == "" {
		format = "text"
	}

	handler, err := newSlogHandler(format, level)
	if err != nil {
		return nil, err
	}

	logger := slog.New(handler)
	slog.SetDefault(logger)

	return logger, nil
}

func newPool(c *cli.Context, logger *slog.Logger) (*pgxpool.Pool, error) {
	pgxLogger := logger.With("module", "pgx")

	pgxLogLevel := c.String("pgx-log-level")
	if pgxLogLevel == "" {
		pgxLogLevel = "none"
	}

	databaseURL := c.String("database-url")
	if databaseURL == "" {
		return nil, errors.New("--database-url is required")
	}

	config, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	config.ConnConfig.Tracer = &tracelog.TraceLog{Logger: pgxslog.NewLogger(pgxLogger), LogLevel: parseTraceLogLevel(pgxLogLevel)}
	config.MaxConns = 10

	return pgxpool.NewWithConfig(context.Background(), config)
}

func newMailer(c *cli.Context, logger *slog.Logger) (backend.Mailer, error) {
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

	mailer := &backend.SMTPMailer{
		ServerAddr: smtpServer + ":" + smtpPort,
		Auth:       auth,
		From:       fromAddr,
		RootURL:    rootURL,
		Logger:     logger.With("module", "mail"),
	}

	return mailer, nil
}

func Serve(c *cli.Context) {
	httpConfig := backend.HTTPConfig{
		ListenAddress: c.String("bind-address"),
		ListenPort:    c.String("port"),
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

	feedUpdater := backend.NewFeedUpdater(pool, logger.With("module", "feedUpdater"))
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
