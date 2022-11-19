package testutil

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/testdb"
)

// InitTestDBManager performs the standard initialization of a *testdb.Manager for ISO Amp. It requires a *testing.M to
// ensure it is only called by TestMain. If something fails it calls os.Exit(1).
func InitTestDBManager(*testing.M) *testdb.Manager {
	manager := &testdb.Manager{
		ResetDB: func(ctx context.Context, conn *pgx.Conn) error {
			_, err := conn.Exec(ctx, `select pgundolog.undo()`)
			return err
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	err := manager.Connect(ctx, fmt.Sprintf("dbname=%s", os.Getenv("TEST_DATABASE")))
	if err != nil {
		fmt.Println("failed to init testdb.Manager:", err)
		os.Exit(1)
	}

	return manager
}
