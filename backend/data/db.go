package data

import (
	"context"
	"fmt"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// NewID is generated before issuing SQL so both stores receive the same identity.
func NewID() string { return uuid.New().String() }

// WithinTx runs a complete persistence operation atomically on the selected backend.
func WithinTx(ctx context.Context, db DB, fn func(DB) error) error {
	if db, ok := db.(interface {
		WithinTx(context.Context, func(DB) error) error
	}); ok {
		return db.WithinTx(ctx, fn)
	}
	if db, ok := db.(interface {
		BeginTx(context.Context, pgx.TxOptions) (pgx.Tx, error)
	}); ok {
		tx, err := db.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return err
		}
		defer tx.Rollback(ctx)
		if err := fn(tx); err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	return fmt.Errorf("database does not support transactions")
}

// DB is the SQL surface used by persistence, implemented by PostgreSQL, JED and dualstore.
type DB interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}
