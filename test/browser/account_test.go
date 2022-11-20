package browser_test

import (
	"context"
	"testing"

	"github.com/jackc/tpr/test/testdata"
	"github.com/stretchr/testify/require"
)

func TestUserChangesPassword(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverInstance := startServer(t)
	db := serverInstance.DB.Connect(t, ctx)
	page := TestBrowserManager.Acquire(t).Page()

	testdata.CreateUser(t, db, ctx, map[string]any{"name": "john", "password": "secret"})
	login(t, ctx, page, serverInstance.Server.URL, "john", "secret")

	page.ClickOn("Account")

	page.FillIn("Existing Password", "secret")
	page.FillIn("New Password", "bigsecret")
	page.FillIn("Password Confirmation", "bigsecret")

	page.AcceptDialog(func() {
		page.ClickOn("Update")
	})

	page.ClickOn("Logout")

	login(t, ctx, page, serverInstance.Server.URL, "john", "bigsecret")

	page.HasContent("p", "No unread")
}

func TestUserFailsToChangePasswordBecauseOfWrongExistingPassword(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverInstance := startServer(t)
	db := serverInstance.DB.Connect(t, ctx)
	page := TestBrowserManager.Acquire(t).Page()

	testdata.CreateUser(t, db, ctx, map[string]any{"name": "john", "password": "secret"})
	login(t, ctx, page, serverInstance.Server.URL, "john", "secret")

	page.ClickOn("Account")

	page.FillIn("Existing Password", "wrong")
	page.FillIn("New Password", "bigsecret")
	page.FillIn("Password Confirmation", "bigsecret")

	page.AcceptDialog(func() {
		page.ClickOn("Update")
	})

	page.ClickOn("Logout")

	login(t, ctx, page, serverInstance.Server.URL, "john", "secret")

	page.HasContent("p", "No unread")
}

func TestUserChangesEmail(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverInstance := startServer(t)
	db := serverInstance.DB.Connect(t, ctx)
	page := TestBrowserManager.Acquire(t).Page()

	testdata.CreateUser(t, db, ctx, map[string]any{"name": "john", "password": "secret"})
	login(t, ctx, page, serverInstance.Server.URL, "john", "secret")

	page.ClickOn("Account")

	page.FillIn("Existing Password", "secret")
	page.FillIn("Email", "john@example.com")

	page.AcceptDialog(func() {
		page.ClickOn("Update")
	})

	page.ClickOn("Feeds")
	page.ClickOn("Account")

	require.Equal(t, "john@example.com", page.Page.MustElement("#email").MustEval("function() { return this.value }").String())
}
