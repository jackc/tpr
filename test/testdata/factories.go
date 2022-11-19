package testdata

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgxutil"
	"github.com/jackc/tpr/backend"
	"github.com/jackc/tpr/backend/data"
	"github.com/stretchr/testify/require"
)

type DB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

func CreateUser(t testing.TB, db DB, ctx context.Context, attrs map[string]any) map[string]any {
	if password, ok := attrs["password"]; ok {
		du := &data.User{}
		backend.SetPassword(du, fmt.Sprint(password))
		delete(attrs, "password")
		attrs["password_digest"] = du.PasswordDigest
		attrs["password_salt"] = du.PasswordSalt
	}

	if _, ok := attrs["name"]; !ok {
		attrs["name"] = "test"
	}

	user, err := pgxutil.Insert(ctx, db, "users", attrs)
	require.NoError(t, err)

	return user
}
