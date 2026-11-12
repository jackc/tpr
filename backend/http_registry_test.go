package backend

import (
	"bytes"
	"context"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jackc/tpr/backend/core"
	"github.com/stretchr/testify/require"
)

func TestHTTPRegistryPolicies(t *testing.T) {
	t.Setenv("TEST_ENDPOINTS", "")
	server, err := NewAppServer(HTTPConfig{}, nil, nil, slog.New(slog.DiscardHandler), slog.LevelWarn)
	require.NoError(t, err)
	inventory := HTTPInventory(false)
	require.Len(t, inventory.Endpoints, 22)
	for _, e := range inventory.Endpoints {
		require.Contains(t, e.Stages, "request-id")
		require.Contains(t, e.Stages, "load-session")
		if e.When != nil {
			require.False(t, e.Enabled)
			require.Equal(t, "test-endpoints", e.When.Name)
		}
		if e.Metadata.Public {
			continue
		}
		require.Contains(t, e.Stages, "require-session-user")
		t.Run(e.Method+e.Path, func(t *testing.T) {
			w := httptest.NewRecorder()
			server.ServeHTTP(w, httptest.NewRequest(e.Method, strings.ReplaceAll(e.Path, "{id}", "1"), nil))
			require.Equal(t, 403, w.Code)
			require.Equal(t, "Bad or missing X-Authentication header", w.Body.String())
			require.NotEmpty(t, w.Header().Get("X-Request-Id"))
		})
	}
	for _, e := range HTTPInventory(true).Endpoints {
		require.True(t, e.Enabled)
	}
	for _, path := range []string{"/api/test/reset-db", "/api/test/users", "/api/test/feeds", "/api/test/items", "/api/test/query"} {
		w := httptest.NewRecorder()
		server.ServeHTTP(w, httptest.NewRequest("POST", path, nil))
		require.Equal(t, 404, w.Code)
	}
	w := httptest.NewRecorder()
	server.ServeHTTP(w, httptest.NewRequest("GET", "/health", nil))
	require.Equal(t, 200, w.Code)
	require.Equal(t, "ok", w.Body.String())
}
func TestHTTPJSONRemainsLenient(t *testing.T) {
	w := httptest.NewRecorder()
	RegisterHandler(w, httptest.NewRequest("POST", "/api/register", bytes.NewBufferString(`{"Name":"reader","Password":"short","extra":true}`)), &environment{})
	require.Equal(t, 422, w.Code)
	require.Equal(t, "password must be at least 8 characters\n", w.Body.String())
}
func TestDeleteSessionCannotRevokeAnotherReader(t *testing.T) {
	pool := newConnPool(t)
	c := core.New(pool, nil, testLogger())
	first, err := core.Register.Call(context.Background(), c, core.Registration{Name: "first", Password: "password"})
	require.NoError(t, err)
	second, err := core.Register.Call(context.Background(), c, core.Registration{Name: "second", Password: "password"})
	require.NoError(t, err)
	server, err := NewAppServer(HTTPConfig{}, pool, nil, testLogger(), slog.LevelWarn)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodDelete, "/api/sessions/"+second.SessionID, nil)
	req.Header.Set("X-Authentication", first.SessionID)
	server.ServeHTTP(httptest.NewRecorder(), req)
	for _, token := range []string{first.SessionID, second.SessionID} {
		req := httptest.NewRequest("GET", "/api/account", nil)
		req.Header.Set("X-Authentication", token)
		w := httptest.NewRecorder()
		server.ServeHTTP(w, req)
		require.Equal(t, 200, w.Code)
	}
	req = httptest.NewRequest(http.MethodDelete, "/api/sessions/"+first.SessionID, nil)
	req.Header.Set("X-Authentication", first.SessionID)
	server.ServeHTTP(httptest.NewRecorder(), req)
	req = httptest.NewRequest("GET", "/api/account", nil)
	req.Header.Set("X-Authentication", first.SessionID)
	w := httptest.NewRecorder()
	server.ServeHTTP(w, req)
	require.Equal(t, 403, w.Code)
}
