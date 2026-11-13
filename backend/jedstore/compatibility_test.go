package jedstore_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/tpr/backend/data"
	"github.com/jackc/tpr/backend/jedstore"
	"github.com/stretchr/testify/require"
)

// Exercise the formerly unsupported syntax independently of Go JSON assembly.
func TestUpstreamSQLCompatibility(t *testing.T) {
	ctx := context.Background()
	db, err := jedstore.Open(filepath.Join(t.TempDir(), "compat.jed"))
	require.NoError(t, err)
	defer db.Close()
	_, err = db.Exec(ctx, `create table clocks(t timestamp with time zone)`)
	require.NoError(t, err)
	user := &data.User{Name: pgtype.Text{String: "reader", Valid: true}, PasswordDigest: []byte{1}, PasswordSalt: []byte{2}}
	id, err := data.CreateUser(ctx, db, user)
	require.NoError(t, err)
	require.NoError(t, data.InsertSubscription(ctx, db, id, "https://example.com/feed"))
	const query = `select coalesce(json_agg(row_to_json(t)), '[]'::json)
 from (
  select feeds.id as feed_id, name, extract(epoch from last_fetch_time::timestamptz(0)) as last_fetch_time
  from feeds join subscriptions on feeds.id=subscriptions.feed_id
  where user_id=$1 order by name limit $2
 ) t`
	var result []byte
	require.NoError(t, db.QueryRow(ctx, query, id, 250).Scan(&result))
	require.Contains(t, string(result), `"name":"https://example.com/feed"`)
	require.NoError(t, db.QueryRow(ctx, query, id, 0).Scan(&result))
	require.JSONEq(t, `[]`, string(result))
}
