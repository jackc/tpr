package browser_test

import (
	"context"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/tpr/test/testdata"
)

func TestUserChangesPassword(t *testing.T) {
	t.Parallel()

	browserTest(t, 30*time.Second, func(ctx context.Context, browser *rod.Browser, appHost string, db *pgx.Conn) {
		testdata.CreateUser(t, db, ctx, map[string]any{"name": "john", "password": "secret"})
		page := browser.MustPage("").Context(ctx)
		login(t, ctx, page, appHost, "john", "secret")

		page.MustElementR("a", "Account").MustClick()
		forID := page.MustElementR("label", "Existing Password").MustAttribute("for")
		page.MustElement("#" + *forID).MustInput("secret")

		forID = page.MustElementR("label", "New Password").MustAttribute("for")
		page.MustElement("#" + *forID).MustInput("bigsecret")

		forID = page.MustElementR("label", "Password Confirmation").MustAttribute("for")
		page.MustElement("#" + *forID).MustInput("bigsecret")

		wait, handle := page.MustHandleDialog()
		go page.MustElementR("input", "Update").MustClick()
		wait()
		handle(true, "")

		page.MustElementR("a", "Logout").MustClick()

		login(t, ctx, page, appHost, "john", "bigsecret")

		page.MustElementR("p", "No unread")
	})
}
