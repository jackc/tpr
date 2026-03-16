package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"os"
	"testing"
	"time"

	"log/slog"

	pgxslog "github.com/mcosta74/pgx-slog"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/tracelog"
	"github.com/jackc/tpr/backend/data"
	"github.com/jackc/tpr/test/testdata"
	"github.com/stretchr/testify/require"
)

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func testLogger() *slog.Logger {
	level := getEnvOrDefault("LOG_LEVEL", "none")
	if level == "none" {
		return slog.New(slog.DiscardHandler)
	}

	var lvl slog.Level
	switch level {
	case "debug":
		lvl = slog.LevelDebug
	case "info":
		lvl = slog.LevelInfo
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelWarn
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: lvl}))
}

var sharedPool *pgxpool.Pool

func newConnPool(t testing.TB) *pgxpool.Pool {
	var err error

	if sharedPool == nil {
		logger := testLogger()
		pgxLogger := logger.With("module", "pgx")

		pgxLogLevel := tracelog.LogLevelInfo
		if pgxLevel := os.Getenv("PGX_LOG_LEVEL"); pgxLevel != "" {
			switch pgxLevel {
			case "debug":
				pgxLogLevel = tracelog.LogLevelDebug
			case "info":
				pgxLogLevel = tracelog.LogLevelInfo
			case "warn":
				pgxLogLevel = tracelog.LogLevelWarn
			case "error":
				pgxLogLevel = tracelog.LogLevelError
			case "none":
				pgxLogLevel = tracelog.LogLevelNone
			}
		}

		databaseURL := getEnvOrDefault("DATABASE_URL", "postgres://tpr:secret@127.0.0.1:5432/tpr_test")
		config, err := pgxpool.ParseConfig(databaseURL)
		require.NoError(t, err)

		config.ConnConfig.Tracer = &tracelog.TraceLog{Logger: pgxslog.NewLogger(pgxLogger), LogLevel: pgxLogLevel}
		config.MaxConns = 10

		pool, err := pgxpool.NewWithConfig(context.Background(), config)
		require.NoError(t, err)

		sharedPool = pool
	}

	err = empty(sharedPool)
	require.NoError(t, err)

	return sharedPool
}

// Empty all data in the entire database
func empty(pool *pgxpool.Pool) error {
	tables := []string{"feeds", "items", "password_resets", "sessions", "subscriptions", "unread_items", "users"}
	for _, table := range tables {
		_, err := pool.Exec(context.Background(), fmt.Sprintf("delete from %s", table))
		if err != nil {
			return err
		}
	}
	return nil
}

func TestExportOPML(t *testing.T) {
	pool := newConnPool(t)

	userID, err := data.CreateUser(context.Background(), pool, &data.User{
		Name:           pgtype.Text{String: "test", Valid: true},
		Email:          pgtype.Text{String: "test@example.com", Valid: true},
		PasswordDigest: []byte("digest"),
		PasswordSalt:   []byte("salt"),
	})
	require.NoError(t, err)

	err = data.InsertSubscription(context.Background(), pool, userID, "http://example.com/feed.rss")
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "http://example.com/", nil)
	require.NoError(t, err)

	env := &environment{pool: pool}
	env.user = &data.User{ID: pgtype.Int4{Int32: userID, Valid: true}, Name: pgtype.Text{String: "test", Valid: true}}

	w := httptest.NewRecorder()

	ExportFeedsHandler(w, req, env)

	if w.Code != 200 {
		t.Fatalf("Expected HTTP status 200, instead received %d", w.Code)
	}

	expected := `<?xml version="1.0" encoding="UTF-8"?>
<opml version="1.0"><head><title>The Pithy Reader Export for test</title></head><body><outline text="http://example.com/feed.rss" title="http://example.com/feed.rss" type="rss" xmlUrl="http://example.com/feed.rss"></outline></body></opml>`

	if w.Body.String() != expected {
		t.Fatalf("Expected:\n%s\nGot:\n%s\n", expected, w.Body.String())
	}
}

func TestGetAccountHandler(t *testing.T) {
	pool := newConnPool(t)
	user := &data.User{
		Name:  pgtype.Text{String: "test", Valid: true},
		Email: pgtype.Text{String: "test@example.com", Valid: true},
	}
	SetPassword(user, "password")

	userID, err := data.CreateUser(context.Background(), pool, user)
	require.NoError(t, err)

	user, err = data.SelectUserByPK(context.Background(), pool, userID)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", "http://example.com/", nil)
	require.NoError(t, err)

	env := &environment{user: user, pool: pool}
	w := httptest.NewRecorder()
	GetAccountHandler(w, req, env)

	if w.Code != 200 {
		t.Fatalf("Expected HTTP status 200, instead received %d", w.Code)
	}

	var resp struct {
		ID    int32  `json:"id"`
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	err = json.NewDecoder(w.Body).Decode(&resp)
	require.NoError(t, err)

	if user.ID.Int32 != resp.ID {
		t.Errorf("Expected id %d, instead received %d", user.ID.Int32, resp.ID)
	}

	if user.Name.String != resp.Name {
		t.Errorf("Expected name %s, instead received %s", user.Name.String, resp.Name)
	}

	if user.Email.String != resp.Email {
		t.Errorf("Expected email %s, instead received %s", user.Email.String, resp.Email)
	}
}

func TestUpdateAccountHandler(t *testing.T) {
	origEmail := "test@example.com"
	origPassword := "password"

	var tests = []struct {
		descr               string
		reqEmail            string
		reqExistingPassword string
		reqNewPassword      string
		respCode            int
		actualEmail         string
		actualPassword      string
	}{
		{
			descr:               "Update email and password",
			reqEmail:            "new@example.com",
			reqExistingPassword: origPassword,
			reqNewPassword:      "bigsecret",
			respCode:            200,
			actualEmail:         "new@example.com",
			actualPassword:      "bigsecret",
		},
		{
			descr:               "Update email",
			reqEmail:            "new@example.com",
			reqExistingPassword: origPassword,
			reqNewPassword:      "",
			respCode:            200,
			actualEmail:         "new@example.com",
			actualPassword:      origPassword,
		},
		{
			descr:               "Deny update of email and password",
			reqEmail:            "new@example.com",
			reqExistingPassword: "WRONG",
			reqNewPassword:      "bigsecret",
			respCode:            422,
			actualEmail:         origEmail,
			actualPassword:      origPassword,
		},
		{
			descr:               "Deny update of email",
			reqEmail:            "new@example.com",
			reqExistingPassword: "WRONG",
			reqNewPassword:      "",
			respCode:            422,
			actualEmail:         origEmail,
			actualPassword:      origPassword,
		},
	}

	for _, tt := range tests {
		pool := newConnPool(t)
		user := &data.User{
			Name:  pgtype.Text{String: "test", Valid: true},
			Email: pgtype.Text{String: origEmail, Valid: true},
		}
		SetPassword(user, origPassword)

		userID, err := data.CreateUser(context.Background(), pool, user)
		if err != nil {
			t.Errorf("%s: repo.CreateUser returned error: %v", tt.descr, err)
			continue
		}

		user, err = data.SelectUserByPK(context.Background(), pool, userID)
		if err != nil {
			t.Errorf("%s: repo.GetUser returned error: %v", tt.descr, err)
			continue
		}

		buf := bytes.NewBufferString(`{
			"email": "` + tt.reqEmail + `",
			"existingPassword": "` + tt.reqExistingPassword + `",
			"newPassword": "` + tt.reqNewPassword + `"
		}`)

		req, err := http.NewRequest("PATCH", "http://example.com/", buf)
		if err != nil {
			t.Errorf("%s: http.NewRequest returned error: %v", tt.descr, err)
			continue
		}

		env := &environment{user: user, pool: pool, logger: testLogger()}
		w := httptest.NewRecorder()
		UpdateAccountHandler(w, req, env)

		if w.Code != tt.respCode {
			t.Errorf("%s: Expected HTTP status %d, instead received %d", tt.descr, tt.respCode, w.Code)
			continue
		}

		user, err = data.SelectUserByPK(context.Background(), pool, userID)
		if err != nil {
			t.Errorf("%s: repo.GetUser returned error: %v", tt.descr, err)
			continue
		}

		if user.Email.String != tt.actualEmail {
			t.Errorf("%s: Expected email %s, instead received %s", tt.descr, tt.actualEmail, user.Email.String)
		}

		if !IsPassword(user, tt.actualPassword) {
			t.Errorf("%s: Expected password to be %s, but it wasn't", tt.descr, tt.actualPassword)
		}
	}
}

func TestRequestPasswordResetHandler(t *testing.T) {

	var tests = []struct {
		descr      string
		mailer     *testMailer
		userEmail  string
		reqEmail   string
		remoteAddr string
		remoteHost string
		sentMailTo string
	}{
		{
			descr:      "Email does not match user",
			mailer:     &testMailer{},
			userEmail:  "test@example.com",
			reqEmail:   "other@example.com",
			remoteAddr: "192.168.0.1:54678",
			remoteHost: "192.168.0.1",
		},
		{
			descr:      "Email matches user",
			mailer:     &testMailer{},
			userEmail:  "test@example.com",
			reqEmail:   "test@example.com",
			remoteAddr: "192.168.0.1:54678",
			remoteHost: "192.168.0.1",
			sentMailTo: "test@example.com",
		},
	}

	for _, tt := range tests {
		pool := newConnPool(t)
		user := &data.User{
			Name:  pgtype.Text{String: "test", Valid: true},
			Email: pgtype.Text{String: tt.userEmail, Valid: true},
		}
		SetPassword(user, "password")

		userID, err := data.CreateUser(context.Background(), pool, user)
		if err != nil {
			t.Errorf("%s: repo.CreateUser returned error: %v", tt.descr, err)
			continue
		}

		buf := bytes.NewBufferString(`{"email": "` + tt.reqEmail + `"}`)

		req, err := http.NewRequest("POST", "http://example.com/", buf)
		if err != nil {
			t.Errorf("%s: http.NewRequest returned error: %v", tt.descr, err)
			continue
		}
		req.RemoteAddr = tt.remoteAddr

		env := &environment{user: user, pool: pool, logger: testLogger(), mailer: tt.mailer}
		w := httptest.NewRecorder()
		RequestPasswordResetHandler(w, req, env)

		if w.Code != 200 {
			t.Errorf("%s: Expected HTTP status %d, instead received %d", tt.descr, 200, w.Code)
			continue
		}

		// Need to reach down pgx because repo interface doesn't need any get
		// interface besides by token, but for this test we need to know the token
		var token string
		err = pool.QueryRow(context.Background(), "select token from password_resets").Scan(&token)
		if err != nil {
			t.Errorf("%s: pool.QueryRow Scan returned error: %v", tt.descr, err)
			continue
		}
		pwr, err := data.SelectPasswordResetByPK(context.Background(), env.pool, token)
		if err != nil {
			t.Errorf("%s: repo.GetPasswordReset returned error: %v", tt.descr, err)
			continue
		}

		if pwr.Email != tt.reqEmail {
			t.Errorf("%s: PasswordReset.Email should be %s, but instead is %v", tt.descr, tt.reqEmail, pwr.Email)
		}
		if pwr.RequestIP.String() != tt.remoteHost {
			t.Errorf("%s: PasswordReset.RequestIP should be %s, but instead is %v", tt.descr, tt.remoteHost, pwr.RequestIP.String())
		}
		if tt.reqEmail == tt.userEmail && userID != pwr.UserID.Int32 {
			t.Errorf("%s: PasswordReset.UserID should be %d, but instead is %v", tt.descr, userID, pwr.UserID)
		}
		if tt.reqEmail != tt.userEmail && pwr.UserID.Valid {
			t.Errorf("%s: PasswordReset.UserID should be nil, but instead is %v", tt.descr, pwr.UserID)
		}

		sentMails := tt.mailer.sentPasswordResetMails
		if tt.sentMailTo == "" {
			if len(sentMails) != 0 {
				t.Errorf("%s: Expected to not send any reset mails, instead sent %d", tt.descr, len(sentMails))
			}
			continue
		}

		if len(sentMails) != 1 {
			t.Errorf("%s: Expected to send 1 reset mail, instead sent %d", tt.descr, len(sentMails))
			continue
		}

		if sentMails[0].to != tt.sentMailTo {
			t.Errorf("%s: Expected to send reset mail to %s, instead sent it to %s", tt.descr, tt.sentMailTo, sentMails[0].to)
		}
		if sentMails[0].token != pwr.Token {
			t.Errorf("%s: Reset mail (%v) and password reset (%v) do not have the same token", tt.descr, sentMails[0].token, pwr.Token)
		}
	}
}

func TestResetPasswordHandlerTokenMatchestValidPasswordReset(t *testing.T) {
	pool := newConnPool(t)
	user := &data.User{
		Name:  pgtype.Text{String: "test", Valid: true},
		Email: pgtype.Text{String: "test@example.com", Valid: true},
	}
	SetPassword(user, "password")

	userID, err := data.CreateUser(context.Background(), pool, user)
	if err != nil {
		t.Fatalf("repo.CreateUser returned error: %v", err)
	}

	requestIP := netip.MustParseAddr("127.0.0.1")

	testdata.CreatePasswordReset(t, pool, context.Background(), map[string]any{
		"token":        "0123456789abcdef",
		"email":        "test@example.com",
		"user_id":      userID,
		"request_time": time.Now(),
		"request_ip":   requestIP,
	})

	buf := bytes.NewBufferString(`{"token": "0123456789abcdef", "password": "bigsecret"}`)

	req, err := http.NewRequest("POST", "http://example.com/", buf)
	if err != nil {
		t.Fatalf("http.NewRequest returned error: %v", err)
	}

	env := &environment{pool: pool}
	w := httptest.NewRecorder()
	ResetPasswordHandler(w, req, env)

	if w.Code != 200 {
		t.Errorf("Expected HTTP status %d, instead received %d", 200, w.Code)
	}

	user, err = data.SelectUserByPK(context.Background(), pool, userID)
	if err != nil {
		t.Fatalf("repo.GetUser returned error: %v", err)
	}

	if !IsPassword(user, "bigsecret") {
		t.Error("Expected password to be changed but it was not")
	}

	var response struct {
		Name      string `json:"name"`
		SessionID string `json:"sessionID"`
	}

	decoder := json.NewDecoder(w.Body)
	if err := decoder.Decode(&response); err != nil {
		t.Errorf("Unable to decode response: %v", err)
	}
}

func TestResetPasswordHandlerTokenMatchestUsedPasswordReset(t *testing.T) {
	pool := newConnPool(t)
	user := &data.User{
		Name:  pgtype.Text{String: "test", Valid: true},
		Email: pgtype.Text{String: "test@example.com", Valid: true},
	}
	SetPassword(user, "password")

	userID, err := data.CreateUser(context.Background(), pool, user)
	if err != nil {
		t.Fatalf("repo.CreateUser returned error: %v", err)
	}

	localhost := netip.MustParseAddr("127.0.0.1")

	testdata.CreatePasswordReset(t, pool, context.Background(), map[string]any{
		"token":           "0123456789abcdef",
		"email":           "test@example.com",
		"user_id":         userID,
		"request_time":    time.Now(),
		"request_ip":      localhost,
		"completion_time": time.Now(),
		"completion_ip":   localhost,
	})

	buf := bytes.NewBufferString(`{"token": "0123456789abcdef", "password": "bigsecret"}`)

	req, err := http.NewRequest("POST", "http://example.com/", buf)
	if err != nil {
		t.Fatalf("http.NewRequest returned error: %v", err)
	}

	env := &environment{pool: pool}
	w := httptest.NewRecorder()
	ResetPasswordHandler(w, req, env)

	if w.Code != 404 {
		t.Errorf("Expected HTTP status %d, instead received %d", 404, w.Code)
	}

	user, err = data.SelectUserByPK(context.Background(), pool, userID)
	if err != nil {
		t.Fatalf("repo.GetUser returned error: %v", err)
	}

	if IsPassword(user, "bigsecret") {
		t.Error("Expected password not to be changed but it was")
	}
}

func TestResetPasswordHandlerTokenMatchestInvalidPasswordReset(t *testing.T) {
	pool := newConnPool(t)

	localhost := netip.MustParseAddr("127.0.0.1")

	testdata.CreatePasswordReset(t, pool, context.Background(), map[string]any{
		"token":        "0123456789abcdef",
		"email":        "test@example.com",
		"request_time": time.Now(),
		"request_ip":   localhost,
	})

	buf := bytes.NewBufferString(`{"token": "0123456789abcdef", "password": "bigsecret"}`)

	req, err := http.NewRequest("POST", "http://example.com/", buf)
	if err != nil {
		t.Fatalf("http.NewRequest returned error: %v", err)
	}

	env := &environment{pool: pool}
	w := httptest.NewRecorder()
	ResetPasswordHandler(w, req, env)

	if w.Code != 404 {
		t.Errorf("Expected HTTP status %d, instead received %d", 404, w.Code)
	}
}

func TestResetPasswordHandlerTokenDoesNotMatchPasswordReset(t *testing.T) {
	pool := newConnPool(t)

	buf := bytes.NewBufferString(`{"token": "0123456789abcdef", "password": "bigsecret"}`)

	req, err := http.NewRequest("POST", "http://example.com/", buf)
	if err != nil {
		t.Fatalf("http.NewRequest returned error: %v", err)
	}

	env := &environment{pool: pool}
	w := httptest.NewRecorder()
	ResetPasswordHandler(w, req, env)

	if w.Code != 404 {
		t.Errorf("Expected HTTP status %d, instead received %d", 404, w.Code)
	}
}
