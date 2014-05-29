package main

import (
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
