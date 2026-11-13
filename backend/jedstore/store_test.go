package jedstore_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/tpr/backend/data"
	"github.com/jackc/tpr/backend/dualstore"
	"github.com/jackc/tpr/backend/jedstore"
	"github.com/stretchr/testify/require"
)

func TestSharedDatabase(t *testing.T) { testStore(t, false) }
func TestDualStore(t *testing.T)      { testStore(t, true) }
func testStore(t *testing.T, dual bool) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "tpr.jed")
	s, err := jedstore.Open(path)
	require.NoError(t, err)
	defer func() {
		if s != nil {
			s.Close()
		}
	}()
	var db data.DB = s
	if dual {
		url := os.Getenv("TEST_DATABASE_URL")
		if url == "" {
			t.Skip("TEST_DATABASE_URL required for dual conformance")
		}
		cfg, err := pgxpool.ParseConfig(url)
		require.NoError(t, err)
		cfg.ConnConfig.Database += "_8"
		p, err := pgxpool.NewWithConfig(ctx, cfg)
		require.NoError(t, err)
		defer p.Close()
		for _, table := range []string{"password_resets", "sessions", "unread_items", "subscriptions", "items", "feeds", "users"} {
			_, err = p.Exec(ctx, "delete from "+table)
			require.NoError(t, err)
		}
		db = dualstore.New(p, s)
	}
	u := &data.User{Name: pgtype.Text{String: "reader", Valid: true}, PasswordDigest: []byte("digest"), PasswordSalt: []byte("salt")}
	id, err := data.CreateUser(ctx, db, u)
	require.NoError(t, err)
	got, err := data.SelectUserByPK(ctx, db, id)
	require.NoError(t, err)
	require.Equal(t, u, got)
	require.NoError(t, data.InsertSubscription(ctx, db, id, "https://example.com/feed"))
	subs, err := data.SelectSubscriptions(ctx, db, id)
	require.NoError(t, err)
	require.Len(t, subs, 1)
	now := time.Now().UTC().Truncate(time.Microsecond)
	require.NoError(t, data.UpdateFeedWithFetchSuccess(ctx, db, subs[0].FeedID.String, &data.ParsedFeed{Name: "Feed", Items: []data.ParsedItem{{Title: "Article", URL: "https://example.com/1"}}}, pgtype.Text{}, now))
	var b bytes.Buffer
	require.NoError(t, data.CopyUnreadItemsAsJSONByUserID(ctx, db, &b, id))
	require.Contains(t, b.String(), "Article")
	b.Reset()
	require.NoError(t, data.CopySubscriptionsForUserAsJSON(ctx, db, &b, id))
	b.Reset()
	require.NoError(t, data.CopyArchivedItemsAsJSONByUserID(ctx, db, &b, id))
	u2 := &data.User{Name: pgtype.Text{String: "second", Valid: true}, PasswordDigest: []byte("digest"), PasswordSalt: []byte("salt")}
	id2, err := data.CreateUser(ctx, db, u2)
	require.NoError(t, err)
	require.NoError(t, data.InsertSubscription(ctx, db, id2, "https://example.com/feed"))
	subs2, err := data.SelectSubscriptions(ctx, db, id2)
	require.NoError(t, err)
	require.Equal(t, subs[0].FeedID, subs2[0].FeedID)
	b.Reset()
	require.NoError(t, data.CopyUnreadItemsAsJSONByUserID(ctx, db, &b, id2))
	require.JSONEq(t, "[]", b.String())
	require.NoError(t, data.DeleteSubscription(ctx, db, id, subs[0].FeedID.String))
	exerciseAuth(t, db)
	exerciseTransactions(t, db)
	if d, ok := db.(*dualstore.Store); ok {
		require.NoError(t, d.Verify(ctx))
	}
	copyDB, err := jedstore.Open(filepath.Join(t.TempDir(), "copy.jed"))
	require.NoError(t, err)
	defer copyDB.Close()
	require.NoError(t, data.CopySnapshot(ctx, db, copyDB))
	checker := dualstore.New(db, copyDB)
	require.NoError(t, checker.Verify(ctx))
	require.ErrorContains(t, data.CopySnapshot(ctx, db, copyDB), "not empty")
	_, err = copyDB.Exec(ctx, "update users set email=$1 where id=$2", "diverged@example.com", id2)
	require.NoError(t, err)
	require.Panics(t, func() { checker.Verify(ctx) })
	require.Panics(t, func() { checker.Exec(ctx, "delete from users where id=$1", id2) })
	require.NoError(t, s.Close())
	s = nil
	reopened, err := jedstore.Open(path)
	require.NoError(t, err)
	defer reopened.Close()
	subs2, err = data.SelectSubscriptions(ctx, reopened, id2)
	require.NoError(t, err)
	require.Len(t, subs2, 7)
}
