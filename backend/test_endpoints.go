package backend

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"sync/atomic"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgxutil"
	log "gopkg.in/inconshreveable/log15.v2"
	"golang.org/x/crypto/scrypt"
)

var counter atomic.Int64

// RegisterTestEndpoints adds test-only endpoints to the API router
// Only call this when TEST_ENDPOINTS environment variable is set
func RegisterTestEndpoints(r chi.Router, pool *pgxpool.Pool, logger log.Logger) {
	r.Post("/test/reset-db", func(w http.ResponseWriter, req *http.Request) {
		ctx := context.Background()

		// Get database name from the pool config
		config := pool.Config()
		dbName := config.ConnConfig.Database

		// Connect as postgres superuser for pgundolog.undo()
		// This requires superuser privileges to set session_replication_role
		connString := fmt.Sprintf("host=%s port=%d dbname=%s user=postgres sslmode=disable",
			config.ConnConfig.Host, config.ConnConfig.Port, dbName)

		conn, err := pgx.Connect(ctx, connString)
		if err != nil {
			logger.Error("Failed to connect as postgres", "error", err)
			http.Error(w, fmt.Sprintf("Failed to connect as postgres: %v", err), 500)
			return
		}
		defer conn.Close(ctx)

		_, err = conn.Exec(ctx, "SELECT pgundolog.undo()")
		if err != nil {
			logger.Error("Failed to reset database", "error", err)
			http.Error(w, fmt.Sprintf("Failed to reset database: %v", err), 500)
			return
		}

		w.WriteHeader(204)
	})

	r.Post("/test/users", func(w http.ResponseWriter, req *http.Request) {
		var attrs map[string]any
		if err := json.NewDecoder(req.Body).Decode(&attrs); err != nil {
			http.Error(w, fmt.Sprintf("Error decoding request: %v", err), 400)
			return
		}

		// Handle password hashing (from testdata.CreateUser)
		if password, ok := attrs["password"]; ok {
			salt := make([]byte, 8)
			if _, err := rand.Read(salt); err != nil {
				http.Error(w, fmt.Sprintf("Failed to generate salt: %v", err), 500)
				return
			}

			digest, err := scrypt.Key([]byte(fmt.Sprint(password)), salt, 16384, 8, 1, 32)
			if err != nil {
				http.Error(w, fmt.Sprintf("Failed to hash password: %v", err), 500)
				return
			}

			delete(attrs, "password")
			attrs["password_digest"] = digest
			attrs["password_salt"] = salt
		}

		// Set default name if not provided
		if _, ok := attrs["name"]; !ok {
			attrs["name"] = "test"
		}

		ctx := context.Background()
		user, err := pgxutil.InsertRowReturning(ctx, pool, "users", attrs, "*", pgx.RowToMap)
		if err != nil {
			logger.Error("Failed to create user", "error", err)
			http.Error(w, fmt.Sprintf("Failed to create user: %v", err), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(user)
	})

	r.Post("/test/feeds", func(w http.ResponseWriter, req *http.Request) {
		var attrs map[string]any
		if err := json.NewDecoder(req.Body).Decode(&attrs); err != nil {
			http.Error(w, fmt.Sprintf("Error decoding request: %v", err), 400)
			return
		}

		if attrs == nil {
			attrs = make(map[string]any)
		}

		// Set defaults (from testdata.CreateFeed)
		n := counter.Add(1)
		if _, ok := attrs["name"]; !ok {
			attrs["name"] = fmt.Sprintf("Feed %v", n)
		}
		if _, ok := attrs["url"]; !ok {
			attrs["url"] = fmt.Sprintf("http://localhost/%v", n)
		}

		ctx := context.Background()
		feed, err := pgxutil.InsertRowReturning(ctx, pool, "feeds", attrs, "*", pgx.RowToMap)
		if err != nil {
			logger.Error("Failed to create feed", "error", err)
			http.Error(w, fmt.Sprintf("Failed to create feed: %v", err), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(feed)
	})

	r.Post("/test/items", func(w http.ResponseWriter, req *http.Request) {
		var attrs map[string]any
		if err := json.NewDecoder(req.Body).Decode(&attrs); err != nil {
			http.Error(w, fmt.Sprintf("Error decoding request: %v", err), 400)
			return
		}

		// Set defaults (from testdata.CreateItem)
		n := counter.Add(1)
		if _, ok := attrs["feed_id"]; !ok {
			// Create a feed if feed_id not provided
			feed, err := pgxutil.InsertRowReturning(context.Background(), pool, "feeds", map[string]any{
				"name": fmt.Sprintf("Feed %v", n),
				"url":  fmt.Sprintf("http://localhost/%v", n),
			}, "*", pgx.RowToMap)
			if err != nil {
				logger.Error("Failed to create feed for item", "error", err)
				http.Error(w, fmt.Sprintf("Failed to create feed: %v", err), 500)
				return
			}
			attrs["feed_id"] = feed["id"]
		}
		if _, ok := attrs["title"]; !ok {
			attrs["title"] = fmt.Sprintf("Title %v", n)
		}
		if _, ok := attrs["url"]; !ok {
			attrs["url"] = fmt.Sprintf("http://localhost/%v", n)
		}

		ctx := context.Background()
		item, err := pgxutil.InsertRowReturning(ctx, pool, "items", attrs, "*", pgx.RowToMap)
		if err != nil {
			logger.Error("Failed to create item", "error", err)
			http.Error(w, fmt.Sprintf("Failed to create item: %v", err), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(item)
	})

	r.Post("/test/query", func(w http.ResponseWriter, req *http.Request) {
		var query struct {
			SQL    string `json:"sql"`
			Params []any  `json:"params"`
		}
		if err := json.NewDecoder(req.Body).Decode(&query); err != nil {
			http.Error(w, fmt.Sprintf("Error decoding request: %v", err), 400)
			return
		}

		ctx := context.Background()
		rows, err := pool.Query(ctx, query.SQL, query.Params...)
		if err != nil {
			logger.Error("Query failed", "error", err, "sql", query.SQL)
			http.Error(w, fmt.Sprintf("Query failed: %v", err), 500)
			return
		}
		defer rows.Close()

		results, err := pgx.CollectRows(rows, pgx.RowToMap)
		if err != nil {
			logger.Error("Failed to collect rows", "error", err)
			http.Error(w, fmt.Sprintf("Failed to collect rows: %v", err), 500)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(results)
	})
}
