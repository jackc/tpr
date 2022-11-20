package browser_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/require"
)

func TestRegisterNewUser(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverInstance := startServer(t)
	db := serverInstance.DB.Connect(t, ctx)
	page := TestBrowserManager.Acquire(t).Page()

	page.MustNavigate(fmt.Sprintf("%s/#login", serverInstance.Server.URL))

	page.ClickOn("Create an account")

	page.FillIn("User name", "joe")
	page.FillIn(`Email \(optional\)`, "joe@example.com")
	page.FillIn("Password", "bigsecret")
	page.FillIn("Password Confirmation", "bigsecret")

	page.ClickOn("Register")

	page.HasContent("body", "No unread items")

	rows, _ := db.Query(ctx, "select count(*) from users")
	userCount, err := pgx.CollectOneRow(rows, pgx.RowTo[int64])
	require.NoError(t, err)
	require.EqualValues(t, 1, userCount)

	rows, _ = db.Query(ctx, "select * from users")
	user, err := pgx.CollectOneRow(rows, pgx.RowToMap)
	require.NoError(t, err)
	require.Equal(t, "joe", user["name"])
	require.Equal(t, "joe@example.com", user["email"])
}

func TestRegisterNewUserWithoutEmail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverInstance := startServer(t)
	db := serverInstance.DB.Connect(t, ctx)
	page := TestBrowserManager.Acquire(t).Page()

	page.MustNavigate(fmt.Sprintf("%s/#login", serverInstance.Server.URL))

	page.ClickOn("Create an account")

	page.FillIn("User name", "joe")
	page.FillIn("Password", "bigsecret")
	page.FillIn("Password Confirmation", "bigsecret")

	page.ClickOn("Register")

	page.HasContent("body", "No unread items")

	rows, _ := db.Query(ctx, "select count(*) from users")
	userCount, err := pgx.CollectOneRow(rows, pgx.RowTo[int64])
	require.NoError(t, err)
	require.EqualValues(t, 1, userCount)

	rows, _ = db.Query(ctx, "select * from users")
	user, err := pgx.CollectOneRow(rows, pgx.RowToMap)
	require.NoError(t, err)
	require.Equal(t, "joe", user["name"])
	require.Equal(t, nil, user["email"])
}
