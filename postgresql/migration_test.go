package postgresql_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"
	"uuid"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestUUIDMigrationRoundTrip(t *testing.T) {
	for _, legacyNames := range []bool{false, true} {
		name := "current constraint names"
		if legacyNames {
			name = "legacy constraint names"
		}
		t.Run(name, func(t *testing.T) { testUUIDMigrationRoundTrip(t, legacyNames) })
	}
}

func testUUIDMigrationRoundTrip(t *testing.T, legacyNames bool) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL required")
	}
	ctx := context.Background()
	cfg, err := pgx.ParseConfig(url)
	require.NoError(t, err)
	cfg.User = "postgres"
	cfg.Password = ""
	conn, err := pgx.ConnectConfig(ctx, cfg)
	require.NoError(t, err)
	defer conn.Close(ctx)
	schema := "migration_" + strings.ReplaceAll(uuid.New().String(), "-", "")
	_, err = conn.Exec(ctx, "create schema "+schema)
	require.NoError(t, err)
	defer conn.Exec(ctx, "drop schema "+schema+" cascade")
	_, err = conn.Exec(ctx, "set search_path="+schema)
	require.NoError(t, err)
	tmpl, err := template.New("migrations").ParseFiles("migrations/func/create_subscription_001.sql")
	require.NoError(t, err)
	// The production template name includes its relative directory.
	function, err := os.ReadFile("migrations/func/create_subscription_001.sql")
	require.NoError(t, err)
	_, err = tmpl.New("func/create_subscription_001.sql").Parse(string(function))
	require.NoError(t, err)
	apply := func(file string, down bool) {
		t.Helper()
		contents, err := os.ReadFile(file)
		require.NoError(t, err)
		parts := strings.Split(string(contents), "---- create above / drop below ----")
		part := parts[0]
		if down {
			part = parts[1]
		}
		m, err := tmpl.New("current").Parse(part)
		require.NoError(t, err)
		var b bytes.Buffer
		require.NoError(t, m.Execute(&b, map[string]string{"app_user": "tpr"}))
		tx, err := conn.Begin(ctx)
		require.NoError(t, err)
		defer tx.Rollback(ctx)
		_, err = tx.Exec(ctx, b.String())
		require.NoError(t, err)
		require.NoError(t, tx.Commit(ctx))
	}
	files, err := filepath.Glob("migrations/*.sql")
	require.NoError(t, err)
	for _, f := range files {
		if strings.HasPrefix(filepath.Base(f), "012_") {
			break
		}
		apply(f, false)
	}
	_, err = conn.Exec(ctx, `insert into users(name,password_digest,password_salt) values ('legacy','digest','salt');
 select create_subscription(1,'https://example.com/feed');
 insert into items(feed_id,title,url) values(1,'old','https://example.com/item');
 insert into unread_items values(1,1,1);
 insert into sessions(id,user_id) values('session',1);
 insert into password_resets(token,email,request_time,user_id) values('known','a@example.com',now(),1),('unknown','b@example.com',now(),null);`)
	require.NoError(t, err)
	if legacyNames {
		_, err = conn.Exec(ctx, `alter table unread_items rename constraint unread_items_user_id_feed_id_fkey to unread_items_user_id_fkey;
   alter table unread_items rename constraint unread_items_feed_id_item_id_fkey to unread_items_feed_id_fkey;`)
		require.NoError(t, err)
	}
	migration := "migrations/012_uuid_identifiers.sql"
	apply(migration, false)
	var count int
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from unread_items u join users r on r.id=u.user_id join items i on i.id=u.item_id join feeds f on f.id=u.feed_id`).Scan(&count))
	require.Equal(t, 1, count)
	var isNull bool
	require.NoError(t, conn.QueryRow(ctx, `select user_id is null from password_resets where token='unknown'`).Scan(&isNull))
	require.True(t, isNull)
	_, err = conn.Exec(ctx, `insert into users(name,password_digest,password_salt) values('new','digest','salt')`)
	require.NoError(t, err)
	apply(migration, true)
	require.NoError(t, conn.QueryRow(ctx, `select id from users where name='legacy'`).Scan(&count))
	require.Equal(t, 1, count)
	require.NoError(t, conn.QueryRow(ctx, `select id from users where name='new'`).Scan(&count))
	require.Equal(t, 2, count)
	_, err = conn.Exec(ctx, `select create_subscription(2,'https://example.com/new')`)
	require.NoError(t, err)
	apply(migration, false)
	require.NoError(t, conn.QueryRow(ctx, `select count(*) from subscriptions s join users u on u.id=s.user_id join feeds f on f.id=s.feed_id`).Scan(&count))
	require.Equal(t, 2, count)
}
