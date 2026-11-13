package jedstore_test

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/tpr/backend/core"
	"github.com/jackc/tpr/backend/data"
	"github.com/stretchr/testify/require"
)

type mailer struct{ token string }

func (m *mailer) SendPasswordResetMail(to, token string) error { m.token = token; return nil }
func exerciseAuth(t *testing.T, db data.DB) {
	ctx := context.Background()
	m := &mailer{}
	c := core.New(db, m, nil)
	session, err := core.Register.Call(ctx, c, core.Registration{Name: "auth", Email: "auth@example.com", Password: "password123"})
	require.NoError(t, err)
	token, err := hex.DecodeString(session.SessionID)
	require.NoError(t, err)
	user, err := data.SelectUserBySessionID(ctx, db, token)
	require.NoError(t, err)
	_, err = core.Register.Call(ctx, c, core.Registration{Name: "AUTH", Email: "auth2@example.com", Password: "password123"})
	require.ErrorContains(t, err, "already taken")
	_, err = core.Register.Call(ctx, c, core.Registration{Name: "auth2", Email: "auth@example.com", Password: "password123"})
	require.ErrorContains(t, err, "already taken")
	_, err = core.CreateSession.Call(ctx, c, core.Credentials{Name: "auth", Password: "wrong"})
	require.ErrorContains(t, err, "Bad user")
	_, err = core.UpdateAccount.Call(core.WithUser(ctx, user), c, core.AccountUpdate{Email: "changed@example.com", ExistingPassword: "password123", NewPassword: "newpassword123"})
	require.NoError(t, err)
	_, err = core.CreateSession.Call(ctx, c, core.Credentials{Name: "auth", Password: "newpassword123"})
	require.NoError(t, err)
	require.NoError(t, data.DeleteSessionByUserID(ctx, db, token, data.NewID()))
	_, err = data.SelectUserBySessionID(ctx, db, token)
	require.NoError(t, err)
	_, err = core.DeleteSession.Call(core.WithUser(ctx, user), c, core.SessionID{ID: session.SessionID})
	require.NoError(t, err)
	_, err = data.SelectUserBySessionID(ctx, db, token)
	require.ErrorIs(t, err, pgx.ErrNoRows)
	_, err = core.RequestPasswordReset.Call(ctx, c, core.PasswordResetRequest{Email: "changed@example.com", RequestIP: netip.MustParseAddr("2001:db8::1")})
	require.NoError(t, err)
	reset, err := data.SelectPasswordResetByPK(ctx, db, m.token)
	require.NoError(t, err)
	require.Equal(t, "2001:db8::1", reset.RequestIP.String())
	_, err = core.ResetPassword.Call(ctx, c, core.PasswordReset{Token: m.token, Password: "resetpassword123"})
	require.NoError(t, err)
	_, err = core.CreateSession.Call(ctx, c, core.Credentials{Name: "auth", Password: "resetpassword123"})
	require.NoError(t, err)
	_, err = core.RequestPasswordReset.Call(ctx, c, core.PasswordResetRequest{Email: "unknown@example.com"})
	require.NoError(t, err)
}

func exerciseTransactions(t *testing.T, db data.DB) {
	ctx := context.Background()
	sentinel := errors.New("rollback")
	err := data.WithinTx(ctx, db, func(tx data.DB) error {
		_, err := data.CreateUser(ctx, tx, &data.User{Name: pgtype.Text{String: "rolledback", Valid: true}, PasswordDigest: []byte{1}, PasswordSalt: []byte{2}})
		if err != nil {
			return err
		}
		return sentinel
	})
	require.ErrorIs(t, err, sentinel)
	_, err = data.SelectUserByName(ctx, db, "rolledback")
	require.ErrorIs(t, err, pgx.ErrNoRows)
	user, err := data.SelectUserByName(ctx, db, "second")
	require.NoError(t, err)
	// Concurrent subscriptions exercise shared-file serialization and consistent IDs.
	var wg sync.WaitGroup
	errs := make(chan error, 6)
	for i := range 6 {
		wg.Go(func() {
			errs <- data.InsertSubscription(ctx, db, user.ID.String, fmt.Sprintf("https://example.com/concurrent/%d", i))
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	stale, err := data.GetFeedsUncheckedSince(ctx, db, time.Now().Add(time.Hour))
	require.NoError(t, err)
	require.NotEmpty(t, stale)
	now := time.Now().UTC().Truncate(time.Microsecond)
	feedID := stale[0].ID
	require.NoError(t, data.UpdateFeedWithFetchFailure(ctx, db, feedID, "failed", now))
	require.NoError(t, data.UpdateFeedWithFetchUnchanged(ctx, db, feedID, now))
	parsed := &data.ParsedFeed{Name: "Updated", Items: []data.ParsedItem{{URL: "https://example.com/new", Title: "New", PublicationTime: pgtype.Timestamptz{Time: now, Valid: true}}}}
	require.NoError(t, data.UpdateFeedWithFetchSuccess(ctx, db, feedID, parsed, pgtype.Text{String: "etag", Valid: true}, now))
	require.NoError(t, data.UpdateFeedWithFetchSuccess(ctx, db, feedID, parsed, pgtype.Text{String: "etag", Valid: true}, now))
	var itemID string
	require.NoError(t, db.QueryRow(ctx, "select id from items where feed_id=$1 and url=$2", feedID, "https://example.com/new").Scan(&itemID))
	require.ErrorIs(t, data.MarkItemRead(ctx, db, data.NewID(), itemID), pgx.ErrNoRows)
	require.NoError(t, data.MarkItemRead(ctx, db, user.ID.String, itemID))
	require.ErrorIs(t, data.MarkItemRead(ctx, db, user.ID.String, itemID), pgx.ErrNoRows)
	// Last-unsubscribe must remove the feed and its items in both stores.
	require.NoError(t, data.DeleteSubscription(ctx, db, user.ID.String, feedID))
	var count int
	require.NoError(t, db.QueryRow(ctx, "select count(*) from items where feed_id=$1", feedID).Scan(&count))
	require.Zero(t, count)
	// Restore the subscription so reopen assertions retain a stable count.
	require.NoError(t, data.InsertSubscription(ctx, db, user.ID.String, stale[0].URL))
	exerciseParameterizedCTE(t, db, user.ID.String)
}

// The CTE and its primary statement use different parameter subsets. Both must
// resolve and bind before the statement writes, including the bare UUID below.
func exerciseParameterizedCTE(t *testing.T, db data.DB, userID string) {
	t.Helper()
	ctx := context.Background()
	const url = "https://example.com/cte"
	require.NoError(t, data.InsertSubscription(ctx, db, userID, url))
	var feedID string
	require.NoError(t, db.QueryRow(ctx, "select id from feeds where url=$1", url).Scan(&feedID))
	itemID := data.NewID()
	tag, err := db.Exec(ctx, `with new_item as (
  insert into items(id,feed_id,url,title,creation_time)
  values($1,$2,$3,$4,$5) returning id,feed_id
 )
 insert into unread_items(user_id,feed_id,item_id)
 select $6,new_item.feed_id,new_item.id from new_item`,
		itemID, feedID, "https://example.com/cte/article", "CTE article", time.Now().UTC().Truncate(time.Microsecond), userID)
	require.NoError(t, err)
	require.EqualValues(t, 1, tag.RowsAffected())
	var got string
	require.NoError(t, db.QueryRow(ctx, "select item_id from unread_items where user_id=$1 and feed_id=$2", userID, feedID).Scan(&got))
	require.Equal(t, itemID, got)
	require.NoError(t, data.DeleteSubscription(ctx, db, userID, feedID))
}
