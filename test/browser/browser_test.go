package browser_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/utils"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/testdb"
	"github.com/jackc/tpr/backend"
	"github.com/jackc/tpr/test/testutil"
	"github.com/stretchr/testify/require"
	log "gopkg.in/inconshreveable/log15.v2"
)

var concurrentChan chan struct{}
var TestDBManager *testdb.Manager
var baseBrowser *rod.Browser

func TestMain(m *testing.M) {
	maxConcurrent := 1
	if n, err := strconv.ParseInt(os.Getenv("MAX_CONCURRENT_BROWSER_TESTS"), 10, 32); err == nil {
		maxConcurrent = int(n)
	}
	if maxConcurrent < 1 {
		fmt.Println("MAX_CONCURRENT_BROWSER_TESTS must be greater than 0")
		os.Exit(1)
	}
	concurrentChan = make(chan struct{}, maxConcurrent)

	TestDBManager = testutil.InitTestDBManager(m)

	baseBrowser = rod.New().MustConnect()

	os.Exit(m.Run())
}

type serverInstanceT struct {
	Server *httptest.Server
	DB     *testdb.DB
}

func startServer(t *testing.T) *serverInstanceT {
	ctx := context.Background()
	db := TestDBManager.AcquireDB(t, ctx)
	handler, err := backend.NewAppServer(backend.HTTPConfig{}, db.PoolConnect(t, ctx), nil, log.New())
	require.NoError(t, err)

	server := httptest.NewServer(handler)

	instance := &serverInstanceT{
		Server: server,
		DB:     db,
	}

	return instance
}

func browserTest(t *testing.T, maxDuration time.Duration, f func(ctx context.Context, browser *rod.Browser, appHost string, db *pgx.Conn)) {
	concurrentChan <- struct{}{}
	defer func() { <-concurrentChan }()

	ctx, cancel := context.WithTimeout(context.Background(), maxDuration)
	defer cancel()

	serverInstance := startServer(t)

	sleeper := func() utils.Sleeper {
		total := time.After(3 * time.Second)

		return func(ctx context.Context) error {
			select {
			case <-time.After(100 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			case <-total:
				return fmt.Errorf("timeout after 3s")
			}
		}
	}

	conn := serverInstance.DB.Connect(t, ctx)

	browser := baseBrowser.MustIncognito().Sleeper(sleeper).WithPanic(func(v interface{}) {
		_, file, line, _ := runtime.Caller(3)
		t.Logf("%v\n    at %s:%d", v, file, line)
		t.FailNow()
	})
	defer browser.MustClose()

	require.NotPanics(t, func() { f(ctx, browser, serverInstance.Server.URL, conn) })
}

// sleepForFlicker waits a little bit to try to solve a flickering test. Obviously, this is a hack. Using this function
// instead of time.Sleep directly so all uses can easily be found and hopefully fixed later.
func sleepForFlicker() {
	time.Sleep(500 * time.Millisecond)
}
