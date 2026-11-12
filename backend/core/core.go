// Package core owns application actions independently of HTTP.
package core

import (
	"context"
	"errors"
	"log/slog"

	"github.com/jackc/manifest/action"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tpr/backend/data"
)

type Mailer interface{ SendPasswordResetMail(to, token string) error }
type Core struct {
	pool    *pgxpool.Pool
	mailer  Mailer
	logger  *slog.Logger
	actions *action.Dispatcher[*Core, ActionMetadata]
}

func New(pool *pgxpool.Pool, mailer Mailer, logger *slog.Logger) *Core {
	if logger == nil {
		logger = slog.Default()
	}
	c := &Core{pool: pool, mailer: mailer, logger: logger}
	c.actions = catalog.NewDispatcher(requireUser)
	return c
}

type userKey struct{}

// WithUser attaches an identity established by a trusted caller or session loader.
func WithUser(ctx context.Context, user *data.User) context.Context {
	return context.WithValue(ctx, userKey{}, user)
}
func User(ctx context.Context) *data.User { u, _ := ctx.Value(userKey{}).(*data.User); return u }

var ErrUnauthenticated = errors.New("authentication required")
var ErrNotFound = errors.New("not found")

type InputError struct{ Message string }

func (e *InputError) Error() string { return e.Message }
func invalid(message string) error  { return &InputError{Message: message} }
func requireUser(next action.Handler[ActionMetadata]) action.Handler[ActionMetadata] {
	return func(ctx context.Context, inv action.Invocation[ActionMetadata]) (any, error) {
		if !inv.Action.Metadata().Public && User(ctx) == nil {
			return nil, ErrUnauthenticated
		}
		return next(ctx, inv)
	}
}

type ActionMetadata struct {
	Public   bool `json:"public"`
	Mutating bool `json:"mutating"`
}
type ActionDef[P, R any] = action.Definition[*Core, ActionMetadata, P, R]
type Action[P, R any] = action.CatalogAction[*Core, ActionMetadata, P, R]

var catalog = action.NewCatalog(action.CatalogOptions[*Core, ActionMetadata]{Dispatcher: func(c *Core) *action.Dispatcher[*Core, ActionMetadata] { return c.actions }})

func Define[P, R any](d ActionDef[P, R]) *Action[P, R]   { return catalog.Define(d) }
func Catalog() []action.AnyAction[*Core, ActionMetadata] { return catalog.Actions() }
func Inventory() action.Inventory[ActionMetadata]        { return catalog.Inventory() }
func (c *Core) InvokeJSON(ctx context.Context, name string, params []byte) (any, error) {
	return catalog.InvokeJSON(ctx, c, name, params)
}

type Empty struct{}
