package browser_test

import (
	"context"
	"testing"

	"github.com/jackc/tpr/test/testdata"
)

func TestUserSubscribesToAFeed(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverInstance := startServer(t)
	db := serverInstance.DB.Connect(t, ctx)
	page := TestBrowserManager.Acquire(t).Page()

	testdata.CreateUser(t, db, ctx, map[string]any{"name": "john", "password": "secret"})
	login(t, ctx, page, serverInstance.Server.URL, "john", "secret")

	page.ClickOn("Feeds")

	page.FillIn("Feed URL", "http://localhost:1234")
	page.ClickOn("Subscribe")

	page.HasContent(".feeds > ul", "http://localhost:1234")
}

func TestUserImportsOPMLFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverInstance := startServer(t)
	db := serverInstance.DB.Connect(t, ctx)
	page := TestBrowserManager.Acquire(t).Page()

	testdata.CreateUser(t, db, ctx, map[string]any{"name": "john", "password": "secret"})
	login(t, ctx, page, serverInstance.Server.URL, "john", "secret")

	page.ClickOn("Feeds")

	page.MustElement(`[type=file]`).MustSetFiles("../testdata/opml.xml")

	page.AcceptDialog(func() {
		page.ClickOn("Import")
	})

	page.HasContent(".feeds > ul", "http://localhost/rss")
	page.HasContent(".feeds > ul", "http://localhost/other/rss")
}
