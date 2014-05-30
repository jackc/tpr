package main

import (
	"bytes"
	"encoding/json"
	"github.com/JackC/box"
	"github.com/JackC/pgx"
	"github.com/vaughan0/go-ini"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newRepository(t *testing.T) repository {
	var err error

	if sharedPgxRepository == nil {
		connPoolConfig := pgx.ConnPoolConfig{MaxConnections: 1, AfterConnect: afterConnect}

		configPath := "../tpr.test.conf"
		file, err := ini.LoadFile(configPath)
		if err != nil {
			t.Fatal(err)
		}

		connPoolConfig.ConnConfig, err = extractConnConfig(file)
		if err != nil {
			t.Fatal(err)
		}

		sharedPgxRepository, err = NewPgxRepository(connPoolConfig)
		if err != nil {
			t.Fatal(err)
		}
	}

	err = sharedPgxRepository.empty()
	if err != nil {
		t.Fatal(err)
	}

	return sharedPgxRepository
}

func TestExportOPML(t *testing.T) {
	repo := newRepository(t)
	userID, err := repo.CreateUser(&User{
		Name:           box.NewString("test"),
		Email:          box.NewString("test@example.com"),
		PasswordDigest: []byte("digest"),
		PasswordSalt:   []byte("salt"),
	})
	if err != nil {
		t.Fatal(err)
	}

	err = repo.CreateSubscription(userID, "http://example.com/feed.rss")
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}

	env := &environment{}
	env.user = &User{ID: box.NewInt32(userID), Name: box.NewString("test")}
	env.repo = repo

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
	repo := newRepository(t)
	user := &User{
		Name:  box.NewString("test"),
		Email: box.NewString("test@example.com"),
	}
	user.SetPassword("password")

	userID, err := repo.CreateUser(user)
	if err != nil {
		t.Fatal(err)
	}

	user, err = repo.GetUser(userID)
	if err != nil {
		t.Fatal(err)
	}

	req, err := http.NewRequest("GET", "http://example.com/", nil)
	if err != nil {
		t.Fatal(err)
	}

	env := &environment{user: user, repo: repo}
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

	decoder := json.NewDecoder(w.Body)
	if err := decoder.Decode(&resp); err != nil {
		t.Fatal(err)
	}

	if user.ID.MustGet() != resp.ID {
		t.Errorf("Expected id %d, instead received %d", user.ID.MustGet(), resp.ID)
	}

	if user.Name.MustGet() != resp.Name {
		t.Errorf("Expected name %s, instead received %s", user.Name.MustGet(), resp.Name)
	}

	if user.Email.MustGet() != resp.Email {
		t.Errorf("Expected email %s, instead received %s", user.Email.MustGet(), resp.Email)
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
		repo := newRepository(t)
		user := &User{
			Name:  box.NewString("test"),
			Email: box.NewString(origEmail),
		}
		user.SetPassword(origPassword)

		userID, err := repo.CreateUser(user)
		if err != nil {
			t.Errorf("%s: repo.CreateUser returned error: %v", tt.descr, err)
			continue
		}

		user, err = repo.GetUser(userID)
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

		env := &environment{user: user, repo: repo}
		w := httptest.NewRecorder()
		UpdateAccountHandler(w, req, env)

		if w.Code != tt.respCode {
			t.Errorf("%s: Expected HTTP status %d, instead received %d", tt.descr, tt.respCode, w.Code)
			continue
		}

		user, err = repo.GetUser(userID)
		if err != nil {
			t.Errorf("%s: repo.GetUser returned error: %v", tt.descr, err)
			continue
		}

		if user.Email.MustGet() != tt.actualEmail {
			t.Errorf("%s: Expected email %s, instead received %s", tt.descr, tt.actualEmail, user.Email.MustGet())
		}

		if !user.IsPassword(tt.actualPassword) {
			t.Errorf("%s: Expected password to be %s, but it wasn't", tt.descr, tt.actualPassword)
		}
	}
}
