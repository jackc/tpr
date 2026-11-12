package backend

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog/v3"
	"github.com/jackc/manifest/httpregistry"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HTTPMetadata struct {
	Public bool `json:"public"`
}

// HTTPInventory uses the serving declarations without connecting to the database.
func HTTPInventory(testEndpoints bool) httpregistry.Inventory[HTTPMetadata] {
	return newHTTPRegistry(nil, nil, slog.New(slog.DiscardHandler), slog.LevelWarn, testEndpoints, "", true).Inventory()
}
func newHTTPRegistry(pool *pgxpool.Pool, mailer Mailer, logger *slog.Logger, logLevel slog.Level, testEndpoints bool, stripPrefix string, full bool) *httpregistry.Registry[HTTPMetadata] {
	compiler := func(spec httpregistry.Spec[HTTPMetadata]) ([]httpregistry.Stage, error) {
		if spec.Metadata.Public {
			return nil, nil
		}
		return []httpregistry.Stage{{Name: "require-session-user", Wrap: func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if userFromContext(r.Context()) == nil {
					w.WriteHeader(403)
					fmt.Fprint(w, "Bad or missing X-Authentication header")
					return
				}
				next.ServeHTTP(w, r)
			})
		}}}, nil
	}
	var stages []httpregistry.Stage
	if full {
		stages = []httpregistry.Stage{
			{Name: "request-id", Wrap: middleware.RequestID},
			{Name: "response-request-id", Wrap: func(next http.Handler) http.Handler {
				return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if reqID := middleware.GetReqID(r.Context()); reqID != "" {
						w.Header().Set(middleware.RequestIDHeader, reqID)
					}
					next.ServeHTTP(w, r)
				})
			}},
			{Name: "load-session", Wrap: loadSession(pool)},
			{Name: "request-log-and-recovery", Wrap: httplog.RequestLogger(logger, &httplog.Options{
				Level:         logLevel,
				RecoverPanics: true,
				LogExtraAttrs: func(req *http.Request, _ string, _ int) []slog.Attr {
					var attrs []slog.Attr
					if reqID := middleware.GetReqID(req.Context()); reqID != "" {
						attrs = append(attrs, slog.String("request_id", reqID))
					}
					if user := userFromContext(req.Context()); user != nil {
						attrs = append(attrs, slog.String("user_id", fmt.Sprintf("%d", user.ID.Int32)))
						attrs = append(attrs, slog.String("username", user.Name.String))
					}
					return attrs
				},
			})},
		}
	}
	registry := httpregistry.New(compiler, stages...)
	if full {
		registry.Handle(httpregistry.Spec[HTTPMetadata]{Method: "GET", Path: "/health", Metadata: HTTPMetadata{Public: true}}, httpregistry.Custom(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			fmt.Fprint(w, "ok")
		})))
	}
	handle := func(method, path string, public bool, h EnvHandlerFunc) {
		registry.Handle(httpregistry.Spec[HTTPMetadata]{Method: method, Path: strings.TrimPrefix(path, stripPrefix), Metadata: HTTPMetadata{Public: public}}, httpregistry.Custom(EnvHandler(pool, mailer, logger, h)))
	}
	handle("POST", "/api/register", true, RegisterHandler)
	handle("POST", "/api/sessions", true, CreateSessionHandler)
	handle("DELETE", "/api/sessions/{id}", false, DeleteSessionHandler)
	handle("POST", "/api/subscriptions", false, CreateSubscriptionHandler)
	handle("DELETE", "/api/subscriptions/{id}", false, DeleteSubscriptionHandler)
	handle("POST", "/api/request_password_reset", true, RequestPasswordResetHandler)
	handle("POST", "/api/reset_password", true, ResetPasswordHandler)
	handle("GET", "/api/feeds", false, GetFeedsHandler)
	handle("POST", "/api/feeds/import", false, ImportFeedsHandler)
	handle("GET", "/api/feeds.xml", false, ExportFeedsHandler)
	handle("GET", "/api/items/unread", false, GetUnreadItemsHandler)
	handle("POST", "/api/items/unread/mark_multiple_read", false, MarkMultipleItemsReadHandler)
	handle("DELETE", "/api/items/unread/{id}", false, MarkItemReadHandler)
	handle("GET", "/api/items/archived", false, GetArchivedItemsHandler)
	handle("GET", "/api/account", false, GetAccountHandler)
	handle("PATCH", "/api/account", false, UpdateAccountHandler)
	registerTestEndpoints(registry, pool, logger, testEndpoints, strings.TrimPrefix("/api", stripPrefix))
	return registry
}
