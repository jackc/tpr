package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/smtp"
	"os"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/jackc/tpr/backend"
	"github.com/jackc/tpr/backend/data"
	pgxslog "github.com/mcosta74/pgx-slog"
	"github.com/urfave/cli/v3"
)

const version = "0.8.1"

// Shared database flags for both commands
var databaseFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "database-url",
		Sources: cli.EnvVars("DATABASE_URL"),
		Usage:   "PostgreSQL connection URL (e.g. postgres://user:pass@host:port/dbname)",
	},
}

// Shared log flags for both commands
var logFlags = []cli.Flag{
	&cli.StringFlag{
		Name:    "log-level",
		Value:   "warn",
		Sources: cli.EnvVars("LOG_LEVEL"),
		Usage:   "Log level (debug, info, warn, error, crit, none)",
	},
	&cli.StringFlag{
		Name:    "log-format",
		Value:   "text",
		Sources: cli.EnvVars("LOG_FORMAT"),
		Usage:   "Log format (text, json, journal)",
	},
	&cli.StringFlag{
		Name:    "pgx-log-level",
		Value:   "warn",
		Sources: cli.EnvVars("PGX_LOG_LEVEL"),
		Usage:   "PGX (database driver) log level",
	},
}

func main() {
	if err := newCommand().Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func newCommand() *cli.Command {
	app := &cli.Command{}
	app.Name = "tpr"
	app.Usage = "The Pithy Reader RSS Aggregator"
	app.Version = version
	app.Authors = []any{"Jack Christensen <jack@jackchristensen.com>"}

	serverFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "bind-address",
			Aliases: []string{"a"},
			Value:   "127.0.0.1",
			Sources: cli.EnvVars("BIND_ADDRESS"),
			Usage:   "address to listen on",
		},
		&cli.StringFlag{
			Name:    "port",
			Aliases: []string{"p"},
			Value:   "8080",
			Sources: cli.EnvVars("PORT"),
			Usage:   "port to listen on",
		},
		&cli.StringFlag{
			Name:    "smtp-server",
			Sources: cli.EnvVars("SMTP_SERVER"),
			Usage:   "SMTP server hostname",
		},
		&cli.StringFlag{
			Name:    "smtp-port",
			Value:   "587",
			Sources: cli.EnvVars("SMTP_PORT"),
			Usage:   "SMTP server port",
		},
		&cli.StringFlag{
			Name:    "smtp-username",
			Sources: cli.EnvVars("SMTP_USERNAME"),
			Usage:   "SMTP username",
		},
		&cli.StringFlag{
			Name:    "smtp-password",
			Sources: cli.EnvVars("SMTP_PASSWORD"),
			Usage:   "SMTP password",
		},
		&cli.StringFlag{
			Name:    "smtp-from",
			Sources: cli.EnvVars("SMTP_FROM"),
			Usage:   "From address for emails",
		},
		&cli.StringFlag{
			Name:    "root-url",
			Sources: cli.EnvVars("ROOT_URL"),
			Usage:   "Root URL for email links (e.g., https://example.com)",
		},
	}
	serverFlags = append(serverFlags, databaseFlags...)
	serverFlags = append(serverFlags, logFlags...)

	resetPasswordFlags := []cli.Flag{
		&cli.StringFlag{
			Name:    "password",
			Aliases: []string{"p"},
			Value:   "",
			Usage:   "password to set",
		},
	}
	resetPasswordFlags = append(resetPasswordFlags, databaseFlags...)
	resetPasswordFlags = append(resetPasswordFlags, logFlags...)

	app.Commands = []*cli.Command{
		{
			Name:        "server",
			Aliases:     []string{"s"},
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

	return app
}

func newLogger(c *cli.Command) (*slog.Logger, error) {
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

func newPool(c *cli.Command, logger *slog.Logger) (*pgxpool.Pool, error) {
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

func newMailer(c *cli.Command, logger *slog.Logger) (backend.Mailer, error) {
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

func Serve(ctx context.Context, c *cli.Command) error {
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

	logLevel, err := parseSlogLevel(c.String("log-level"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	server, err := backend.NewAppServer(httpConfig, pool, mailer, logger, logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Could not create web server: %v\n", err)
		os.Exit(1)
	}

	err = server.Serve()
	if err != nil {
		os.Stderr.WriteString("Could not start web server!\n")
		os.Exit(1)
	}
	return nil
}

func ResetPassword(ctx context.Context, c *cli.Command) error {
	if c.Args().Len() != 1 {
		if err := cli.ShowCommandHelp(ctx, c.Root(), c.Name); err != nil {
			return err
		}
		return cli.Exit("expected exactly one username", 1)
	}

	name := c.Args().Get(0)

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
	return nil
}
