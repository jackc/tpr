package browser_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-rod/rod/lib/input"
	"github.com/jackc/pgxutil"
	"github.com/jackc/tpr/test/testdata"
	"github.com/stretchr/testify/require"
)

func TestUserMarksAllItemsRead(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverInstance := startServer(t)
	db := serverInstance.DB.Connect(t, ctx)
	page := TestBrowserManager.Acquire(t).Page()

	user := testdata.CreateUser(t, db, ctx, map[string]any{"name": "john", "password": "secret"})
	feed := testdata.CreateFeed(t, db, ctx, nil)
	_, err := pgxutil.Insert(ctx, db, "subscriptions", map[string]any{"user_id": user["id"], "feed_id": feed["id"]})
	require.NoError(t, err)
	beforeItem := testdata.CreateItem(t, db, ctx, map[string]any{
		"feed_id":          feed["id"],
		"title":            "First Post",
		"publication_time": time.Date(2014, 2, 6, 10, 34, 51, 0, time.Local),
	})
	_, err = pgxutil.Insert(ctx, db, "unread_items", map[string]any{"user_id": user["id"], "feed_id": feed["id"], "item_id": beforeItem["id"]})
	require.NoError(t, err)

	login(t, ctx, page, serverInstance.Server.URL, "john", "secret")

	page.HasContent("body", "First Post")
	page.HasContent("body", "February 6th, 2014 at 10:34 am")

	// After user is viewing unread items then add another
	afterItem := testdata.CreateItem(t, db, ctx, map[string]any{
		"feed_id": feed["id"],
		"title":   "Second Post",
	})
	_, err = pgxutil.Insert(ctx, db, "unread_items", map[string]any{"user_id": user["id"], "feed_id": feed["id"], "item_id": afterItem["id"]})
	require.NoError(t, err)

	page.ClickOn("Mark All Read")

	page.HasContent("body", "Second Post")

	page.ClickOn("Mark All Read")

	page.HasContent("body", "Refresh")

	// Add another item
	anotherItem := testdata.CreateItem(t, db, ctx, map[string]any{
		"feed_id": feed["id"],
		"title":   "Third Post",
	})
	_, err = pgxutil.Insert(ctx, db, "unread_items", map[string]any{"user_id": user["id"], "feed_id": feed["id"], "item_id": anotherItem["id"]})
	require.NoError(t, err)

	page.ClickOn("Refresh")

	page.HasContent("body", "Third Post")
}

func TestUserUsesKeyboardShortcuts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverInstance := startServer(t)
	db := serverInstance.DB.Connect(t, ctx)
	page := TestBrowserManager.Acquire(t).Page()

	user := testdata.CreateUser(t, db, ctx, map[string]any{"name": "john", "password": "secret"})
	feed := testdata.CreateFeed(t, db, ctx, nil)
	_, err := pgxutil.Insert(ctx, db, "subscriptions", map[string]any{"user_id": user["id"], "feed_id": feed["id"]})
	require.NoError(t, err)
	item := testdata.CreateItem(t, db, ctx, map[string]any{
		"feed_id":          feed["id"],
		"title":            "First Post",
		"publication_time": time.Now().Add(-5 * time.Second),
	})
	_, err = pgxutil.Insert(ctx, db, "unread_items", map[string]any{"user_id": user["id"], "feed_id": feed["id"], "item_id": item["id"]})
	require.NoError(t, err)
	item = testdata.CreateItem(t, db, ctx, map[string]any{
		"feed_id":          feed["id"],
		"title":            "Second Post",
		"publication_time": time.Now(),
	})
	_, err = pgxutil.Insert(ctx, db, "unread_items", map[string]any{"user_id": user["id"], "feed_id": feed["id"], "item_id": item["id"]})
	require.NoError(t, err)

	login(t, ctx, page, serverInstance.Server.URL, "john", "secret")

	page.HasContent(".selected", "First Post")
	page.DoesNotHaveContent(".selected", "Second Post")

	page.Page.Keyboard.MustType('j')

	// The second item is selected and the first is not
	page.DoesNotHaveContent(".selected", "First Post")
	page.HasContent(".selected", "Second Post")

	page.ClickOn("Feeds")
	page.ClickOn("Home")

	// After reloading the page the first post should no longer be visible, but the second should
	page.DoesNotHaveContent("body", "First Post")
	page.HasContent("body", "Second Post")

	// Press Shift+a
	err = page.Page.Keyboard.Press(input.ShiftLeft)
	require.NoError(t, err)
	err = page.Page.Keyboard.Press(input.KeyA)
	require.NoError(t, err)

	err = page.Page.Keyboard.Release(input.ShiftLeft)
	require.NoError(t, err)
	err = page.Page.Keyboard.Release(input.KeyA)
	require.NoError(t, err)

	// After marking all read the second post is no longer visible
	page.DoesNotHaveContent("body", "Second Post")
}

func TestUserLooksAtArchivedPosts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	serverInstance := startServer(t)
	db := serverInstance.DB.Connect(t, ctx)
	page := TestBrowserManager.Acquire(t).Page()

	user := testdata.CreateUser(t, db, ctx, map[string]any{"name": "john", "password": "secret"})
	feed := testdata.CreateFeed(t, db, ctx, nil)
	_, err := pgxutil.Insert(ctx, db, "subscriptions", map[string]any{"user_id": user["id"], "feed_id": feed["id"]})
	require.NoError(t, err)
	beforeItem := testdata.CreateItem(t, db, ctx, map[string]any{
		"feed_id":          feed["id"],
		"title":            "First Post",
		"publication_time": time.Date(2014, 2, 6, 10, 34, 51, 0, time.Local),
	})
	_, err = pgxutil.Insert(ctx, db, "unread_items", map[string]any{"user_id": user["id"], "feed_id": feed["id"], "item_id": beforeItem["id"]})
	require.NoError(t, err)

	login(t, ctx, page, serverInstance.Server.URL, "john", "secret")

	page.HasContent("body", "First Post")
	page.HasContent("body", "February 6th, 2014 at 10:34 am")

	page.ClickOn("Mark All Read")

	page.HasContent("body", "Refresh")

	page.ClickOn("Archive")

	page.HasContent("body", "First Post")
}
