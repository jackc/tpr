package browser_test

import (
	"context"
	"testing"

	"github.com/jackc/tpr/test/testdata"
	"github.com/stretchr/testify/require"
)

func TestUserWithInvalidSessionIsLoggedOut(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverInstance := startServer(t)
	db := serverInstance.DB.Connect(t, ctx)
	page := TestBrowserManager.Acquire(t).Page()

	testdata.CreateUser(t, db, ctx, map[string]any{"name": "john", "password": "secret"})
	login(t, ctx, page, serverInstance.Server.URL, "john", "secret")

	page.HasContent("body", "Refresh")

	_, err := db.Exec(ctx, "delete from sessions")
	require.NoError(t, err)

	page.ClickOn("Feeds")

	page.HasContent("label", "User name")
	page.HasContent("label", "Password")
}
