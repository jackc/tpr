package browser_test

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"strconv"
	"testing"

	"github.com/jackc/testdb"
	"github.com/jackc/tpr/backend"
	"github.com/jackc/tpr/test/testbrowser"
	"github.com/jackc/tpr/test/testutil"
	"github.com/stretchr/testify/require"
	log "gopkg.in/inconshreveable/log15.v2"
)

var concurrentChan chan struct{}
var TestDBManager *testdb.Manager
var TestBrowserManager *testbrowser.Manager

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

	var err error
	TestBrowserManager, err = testbrowser.NewManager(testbrowser.ManagerConfig{})
	if err != nil {
		fmt.Println("Failed to initialize TestBrowserManager")
		os.Exit(1)
	}

	os.Exit(m.Run())
}

type serverInstanceT struct {
	Server *httptest.Server
	DB     *testdb.DB
}

func startServer(t *testing.T) *serverInstanceT {
	ctx := context.Background()
	db := TestDBManager.AcquireDB(t, ctx)
	handler, err := backend.NewAppServer(backend.HTTPConfig{StaticURL: "http://127.0.0.1:5173/"}, db.PoolConnect(t, ctx), nil, log.New())
	require.NoError(t, err)

	server := httptest.NewServer(handler)

	instance := &serverInstanceT{
		Server: server,
		DB:     db,
	}

	return instance
}

func login(t *testing.T, ctx context.Context, page *testbrowser.Page, appHost, email, password string) {
	page.MustNavigate(fmt.Sprintf("%s/#login", appHost))

	page.FillIn("User name", email)
	page.FillIn("Password", password)
	page.ClickOn("Login")
}
