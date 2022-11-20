package testdata

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgxutil"
	"github.com/jackc/tpr/backend"
	"github.com/jackc/tpr/backend/data"
	"github.com/stretchr/testify/require"
)

var counter atomic.Int64

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

func CreateFeed(t testing.TB, db DB, ctx context.Context, attrs map[string]any) map[string]any {
	n := counter.Add(1)

	if attrs == nil {
		attrs = make(map[string]any)
	}

	if _, ok := attrs["name"]; !ok {
		attrs["name"] = fmt.Sprintf("Feed %v", n)
	}
	if _, ok := attrs["url"]; !ok {
		attrs["url"] = fmt.Sprintf("http://localhost/%v", n)
	}

	feed, err := pgxutil.Insert(ctx, db, "feeds", attrs)
	require.NoError(t, err)

	return feed
}

func CreateItem(t testing.TB, db DB, ctx context.Context, attrs map[string]any) map[string]any {
	n := counter.Add(1)

	if _, ok := attrs["feed_id"]; !ok {
		attrs["feed_id"] = CreateFeed(t, db, ctx, nil)["id"]
	}
	if _, ok := attrs["title"]; !ok {
		attrs["title"] = fmt.Sprintf("Title %v", n)
	}
	if _, ok := attrs["url"]; !ok {
		attrs["url"] = fmt.Sprintf("http://localhost/%v", n)
	}

	item, err := pgxutil.Insert(ctx, db, "items", attrs)
	require.NoError(t, err)

	return item
}
