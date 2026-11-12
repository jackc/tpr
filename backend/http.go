package backend

import (
	"context"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"

	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tpr/backend/core"
	"github.com/jackc/tpr/backend/data"
)

type HTTPConfig struct {
	ListenAddress string
	ListenPort    string
}

func userFromContext(ctx context.Context) *data.User { return core.User(ctx) }

func loadSession(pool *pgxpool.Pool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := getUserFromSession(r, pool)
			if user != nil {
				ctx := core.WithUser(r.Context(), user)
				r = r.WithContext(ctx)
			}
			next.ServeHTTP(w, r)
		})
	}
}

type EnvHandlerFunc func(w http.ResponseWriter, req *http.Request, env *environment)

func EnvHandler(pool *pgxpool.Pool, mailer Mailer, logger *slog.Logger, f EnvHandlerFunc) http.Handler {
	c := core.New(pool, mailer, logger)
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		user := userFromContext(req.Context())
		env := &environment{user: user, pool: pool, mailer: mailer, logger: logger, core: c}
		f(w, req, env)
	})
}

type AppServer struct {
	handler    http.Handler
	httpConfig HTTPConfig
	server     *http.Server
}

func NewAppServer(httpConfig HTTPConfig, pool *pgxpool.Pool, mailer Mailer, logger *slog.Logger, logLevel slog.Level) (*AppServer, error) {
	r := newHTTPRegistry(pool, mailer, logger, logLevel, os.Getenv("TEST_ENDPOINTS") == "true", "", true).Handler()

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
	core   *core.Core
	user   *data.User
	pool   *pgxpool.Pool
	logger *slog.Logger
	mailer Mailer
}

func NewAPIHandler(pool *pgxpool.Pool, mailer Mailer, logger *slog.Logger) http.Handler {
	return newHTTPRegistry(pool, mailer, logger, slog.LevelWarn, os.Getenv("TEST_ENDPOINTS") == "true", "/api", false).Handler()
}

func getUserFromSession(req *http.Request, pool *pgxpool.Pool) *data.User {
	token := req.Header.Get("X-Authentication")
	if token == "" {
		token = req.FormValue("session")
	}

	if token == "" {
		return nil
	}
	var sessionID []byte
	sessionID, err := hex.DecodeString(token)
	if err != nil {
		return nil
	}

	// TODO - this could be an error from no records found -- or the connection could be dead or we could have a syntax error...
	user, err := data.SelectUserBySessionID(req.Context(), pool, sessionID)
	if err != nil {
		return nil
	}

	return user
}
