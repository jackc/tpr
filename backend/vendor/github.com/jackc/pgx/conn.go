package pgx

import (
	"context"
	"crypto/md5"
	"crypto/tls"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/chunkreader"
	"github.com/jackc/pgx/pgtype"
)

const (
	connStatusUninitialized = iota
	connStatusClosed
	connStatusIdle
	connStatusBusy
)

// minimalConnInfo has just enough static type information to establish the
// connection and retrieve the type data.
var minimalConnInfo *pgtype.ConnInfo

func init() {
	minimalConnInfo = pgtype.NewConnInfo()
	minimalConnInfo.InitializeDataTypes(map[string]pgtype.Oid{
		"int4": pgtype.Int4Oid,
		"name": pgtype.NameOid,
		"oid":  pgtype.OidOid,
		"text": pgtype.TextOid,
	})
}

// DialFunc is a function that can be used to connect to a PostgreSQL server
type DialFunc func(network, addr string) (net.Conn, error)

// ConnConfig contains all the options used to establish a connection.
type ConnConfig struct {
	Host              string // host (e.g. localhost) or path to unix domain socket directory (e.g. /private/tmp)
	Port              uint16 // default: 5432
	Database          string
	User              string // default: OS user name
	Password          string
	TLSConfig         *tls.Config // config for TLS connection -- nil disables TLS
	UseFallbackTLS    bool        // Try FallbackTLSConfig if connecting with TLSConfig fails. Used for preferring TLS, but allowing unencrypted, or vice-versa
	FallbackTLSConfig *tls.Config // config for fallback TLS connection (only used if UseFallBackTLS is true)-- nil disables TLS
	Logger            Logger
	LogLevel          int
	Dial              DialFunc
	RuntimeParams     map[string]string // Run-time parameters to set on connection as session default values (e.g. search_path or application_name)
}

func (cc *ConnConfig) networkAddress() (network, address string) {
	network = "tcp"
	address = fmt.Sprintf("%s:%d", cc.Host, cc.Port)
	// See if host is a valid path, if yes connect with a socket
	if _, err := os.Stat(cc.Host); err == nil {
		// For backward compatibility accept socket file paths -- but directories are now preferred
		network = "unix"
		address = cc.Host
		if !strings.Contains(address, "/.s.PGSQL.") {
			address = filepath.Join(address, ".s.PGSQL.") + strconv.FormatInt(int64(cc.Port), 10)
		}
	}

	return network, address
}

// Conn is a PostgreSQL connection handle. It is not safe for concurrent usage.
// Use ConnPool to manage access to multiple database connections from multiple
// goroutines.
type Conn struct {
	conn               net.Conn  // the underlying TCP or unix domain socket connection
	lastActivityTime   time.Time // the last time the connection was used
	wbuf               [1024]byte
	writeBuf           WriteBuf
	pid                int32             // backend pid
	secretKey          int32             // key to use to send a cancel query message to the server
	RuntimeParams      map[string]string // parameters that have been reported by the server
	config             ConnConfig        // config used when establishing this connection
	txStatus           byte
	preparedStatements map[string]*PreparedStatement
	channels           map[string]struct{}
	notifications      []*Notification
	logger             Logger
	logLevel           int
	mr                 msgReader
	fp                 *fastpath
	poolResetCount     int
	preallocatedRows   []Rows

	status       int32 // One of connStatus* constants
	causeOfDeath error

	readyForQuery         bool // connection has received ReadyForQuery message since last query was sent
	cancelQueryInProgress int32
	cancelQueryCompleted  chan struct{}

	// context support
	ctxInProgress bool
	doneChan      chan struct{}
	closedChan    chan error

	ConnInfo *pgtype.ConnInfo
}

// PreparedStatement is a description of a prepared statement
type PreparedStatement struct {
	Name              string
	SQL               string
	FieldDescriptions []FieldDescription
	ParameterOids     []pgtype.Oid
}

// PrepareExOptions is an option struct that can be passed to PrepareEx
type PrepareExOptions struct {
	ParameterOids []pgtype.Oid
}

// Notification is a message received from the PostgreSQL LISTEN/NOTIFY system
type Notification struct {
	PID     int32  // backend pid that sent the notification
	Channel string // channel from which notification was received
	Payload string
}

// CommandTag is the result of an Exec function
type CommandTag string

// RowsAffected returns the number of rows affected. If the CommandTag was not
// for a row affecting command (such as "CREATE TABLE") then it returns 0
func (ct CommandTag) RowsAffected() int64 {
	s := string(ct)
	index := strings.LastIndex(s, " ")
	if index == -1 {
		return 0
	}
	n, _ := strconv.ParseInt(s[index+1:], 10, 64)
	return n
}

// Identifier a PostgreSQL identifier or name. Identifiers can be composed of
// multiple parts such as ["schema", "table"] or ["table", "column"].
type Identifier []string

// Sanitize returns a sanitized string safe for SQL interpolation.
func (ident Identifier) Sanitize() string {
	parts := make([]string, len(ident))
	for i := range ident {
		parts[i] = `"` + strings.Replace(ident[i], `"`, `""`, -1) + `"`
	}
	return strings.Join(parts, ".")
}

// ErrNoRows occurs when rows are expected but none are returned.
var ErrNoRows = errors.New("no rows in result set")

// ErrNotificationTimeout occurs when WaitForNotification times out.
var ErrNotificationTimeout = errors.New("notification timeout")

// ErrDeadConn occurs on an attempt to use a dead connection
var ErrDeadConn = errors.New("conn is dead")

// ErrTLSRefused occurs when the connection attempt requires TLS and the
// PostgreSQL server refuses to use TLS
var ErrTLSRefused = errors.New("server refused TLS connection")

// ErrConnBusy occurs when the connection is busy (for example, in the middle of
// reading query results) and another action is attempts.
var ErrConnBusy = errors.New("conn is busy")

// ErrInvalidLogLevel occurs on attempt to set an invalid log level.
var ErrInvalidLogLevel = errors.New("invalid log level")

// ProtocolError occurs when unexpected data is received from PostgreSQL
type ProtocolError string

func (e ProtocolError) Error() string {
	return string(e)
}

// Connect establishes a connection with a PostgreSQL server using config.
// config.Host must be specified. config.User will default to the OS user name.
// Other config fields are optional.
func Connect(config ConnConfig) (c *Conn, err error) {
	return connect(config, minimalConnInfo)
}

func connect(config ConnConfig, connInfo *pgtype.ConnInfo) (c *Conn, err error) {
	c = new(Conn)

	c.config = config
	c.ConnInfo = connInfo

	if c.config.LogLevel != 0 {
		c.logLevel = c.config.LogLevel
	} else {
		// Preserve pre-LogLevel behavior by defaulting to LogLevelDebug
		c.logLevel = LogLevelDebug
	}
	c.logger = c.config.Logger
	c.mr.log = c.log
	c.mr.shouldLog = c.shouldLog

	if c.config.User == "" {
		user, err := user.Current()
		if err != nil {
			return nil, err
		}
		c.config.User = user.Username
		if c.shouldLog(LogLevelDebug) {
			c.log(LogLevelDebug, "Using default connection config", "User", c.config.User)
		}
	}

	if c.config.Port == 0 {
		c.config.Port = 5432
		if c.shouldLog(LogLevelDebug) {
			c.log(LogLevelDebug, "Using default connection config", "Port", c.config.Port)
		}
	}

	network, address := c.config.networkAddress()
	if c.config.Dial == nil {
		c.config.Dial = (&net.Dialer{KeepAlive: 5 * time.Minute}).Dial
	}

	if c.shouldLog(LogLevelInfo) {
		c.log(LogLevelInfo, fmt.Sprintf("Dialing PostgreSQL server at %s address: %s", network, address))
	}
	err = c.connect(config, network, address, config.TLSConfig)
	if err != nil && config.UseFallbackTLS {
		if c.shouldLog(LogLevelInfo) {
			c.log(LogLevelInfo, fmt.Sprintf("Connect with TLSConfig failed, trying FallbackTLSConfig: %v", err))
		}
		err = c.connect(config, network, address, config.FallbackTLSConfig)
	}

	if err != nil {
		if c.shouldLog(LogLevelError) {
			c.log(LogLevelError, fmt.Sprintf("Connect failed: %v", err))
		}
		return nil, err
	}

	return c, nil
}

func (c *Conn) connect(config ConnConfig, network, address string, tlsConfig *tls.Config) (err error) {
	c.conn, err = c.config.Dial(network, address)
	if err != nil {
		return err
	}
	defer func() {
		if c != nil && err != nil {
			c.conn.Close()
			atomic.StoreInt32(&c.status, connStatusClosed)
		}
	}()

	c.RuntimeParams = make(map[string]string)
	c.preparedStatements = make(map[string]*PreparedStatement)
	c.channels = make(map[string]struct{})
	atomic.StoreInt32(&c.status, connStatusIdle)
	c.lastActivityTime = time.Now()
	c.cancelQueryCompleted = make(chan struct{}, 1)
	c.doneChan = make(chan struct{})
	c.closedChan = make(chan error)

	if tlsConfig != nil {
		if c.shouldLog(LogLevelDebug) {
			c.log(LogLevelDebug, "Starting TLS handshake")
		}
		if err := c.startTLS(tlsConfig); err != nil {
			return err
		}
	}

	c.mr.cr = chunkreader.NewChunkReader(c.conn)

	msg := newStartupMessage()

	// Default to disabling TLS renegotiation.
	//
	// Go does not support (https://github.com/golang/go/issues/5742)
	// PostgreSQL recommends disabling (http://www.postgresql.org/docs/9.4/static/runtime-config-connection.html#GUC-SSL-RENEGOTIATION-LIMIT)
	if tlsConfig != nil {
		msg.options["ssl_renegotiation_limit"] = "0"
	}

	// Copy default run-time params
	for k, v := range config.RuntimeParams {
		msg.options[k] = v
	}

	msg.options["user"] = c.config.User
	if c.config.Database != "" {
		msg.options["database"] = c.config.Database
	}

	if err = c.txStartupMessage(msg); err != nil {
		return err
	}

	for {
		var t byte
		var r *msgReader
		t, r, err = c.rxMsg()
		if err != nil {
			return err
		}

		switch t {
		case backendKeyData:
			c.rxBackendKeyData(r)
		case authenticationX:
			if err = c.rxAuthenticationX(r); err != nil {
				return err
			}
		case readyForQuery:
			c.rxReadyForQuery(r)
			if c.shouldLog(LogLevelInfo) {
				c.log(LogLevelInfo, "Connection established")
			}

			// Replication connections can't execute the queries to
			// populate the c.PgTypes and c.pgsqlAfInet
			if _, ok := msg.options["replication"]; ok {
				return nil
			}

			if c.ConnInfo == minimalConnInfo {
				err = c.initConnInfo()
				if err != nil {
					return err
				}
			}

			return nil
		default:
			if err = c.processContextFreeMsg(t, r); err != nil {
				return err
			}
		}
	}
}

func (c *Conn) initConnInfo() error {
	nameOids := make(map[string]pgtype.Oid, 256)

	rows, err := c.Query(`select t.oid, t.typname
from pg_type t
left join pg_type base_type on t.typelem=base_type.oid
where (
	  t.typtype in('b', 'p', 'r')
	  and (base_type.oid is null or base_type.typtype in('b', 'p', 'r'))
	)`)
	if err != nil {
		return err
	}

	for rows.Next() {
		var oid pgtype.Oid
		var name pgtype.Text
		if err := rows.Scan(&oid, &name); err != nil {
			return err
		}

		nameOids[name.String] = oid
	}

	if rows.Err() != nil {
		return rows.Err()
	}

	c.ConnInfo = pgtype.NewConnInfo()
	c.ConnInfo.InitializeDataTypes(nameOids)
	return nil
}

// PID returns the backend PID for this connection.
func (c *Conn) PID() int32 {
	return c.pid
}

// Close closes a connection. It is safe to call Close on a already closed
// connection.
func (c *Conn) Close() (err error) {
	for {
		status := atomic.LoadInt32(&c.status)
		if status < connStatusIdle {
			return nil
		}
		if atomic.CompareAndSwapInt32(&c.status, status, connStatusClosed) {
			break
		}
	}

	defer func() {
		c.conn.Close()
		c.die(errors.New("Closed"))
		if c.shouldLog(LogLevelInfo) {
			c.log(LogLevelInfo, "Closed connection")
		}
	}()

	err = c.conn.SetDeadline(time.Time{})
	if err != nil && c.shouldLog(LogLevelWarn) {
		c.log(LogLevelWarn, "Failed to clear deadlines to send close message", "err", err)
		return err
	}

	_, err = c.conn.Write([]byte{'X', 0, 0, 0, 4})
	if err != nil && c.shouldLog(LogLevelWarn) {
		c.log(LogLevelWarn, "Failed to send terminate message", "err", err)
		return err
	}

	err = c.conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	if err != nil && c.shouldLog(LogLevelWarn) {
		c.log(LogLevelWarn, "Failed to set read deadline to finish closing", "err", err)
		return err
	}

	_, err = c.conn.Read(make([]byte, 1))
	if err != io.EOF {
		return err
	}

	return nil
}

// ParseURI parses a database URI into ConnConfig
//
// Query parameters not used by the connection process are parsed into ConnConfig.RuntimeParams.
func ParseURI(uri string) (ConnConfig, error) {
	var cp ConnConfig

	url, err := url.Parse(uri)
	if err != nil {
		return cp, err
	}

	if url.User != nil {
		cp.User = url.User.Username()
		cp.Password, _ = url.User.Password()
	}

	parts := strings.SplitN(url.Host, ":", 2)
	cp.Host = parts[0]
	if len(parts) == 2 {
		p, err := strconv.ParseUint(parts[1], 10, 16)
		if err != nil {
			return cp, err
		}
		cp.Port = uint16(p)
	}
	cp.Database = strings.TrimLeft(url.Path, "/")

	err = configSSL(url.Query().Get("sslmode"), &cp)
	if err != nil {
		return cp, err
	}

	ignoreKeys := map[string]struct{}{
		"sslmode": {},
	}

	cp.RuntimeParams = make(map[string]string)

	for k, v := range url.Query() {
		if _, ok := ignoreKeys[k]; ok {
			continue
		}

		cp.RuntimeParams[k] = v[0]
	}
	if cp.Password == "" {
		pgpass(&cp)
	}
	return cp, nil
}

var dsnRegexp = regexp.MustCompile(`([a-zA-Z_]+)=((?:"[^"]+")|(?:[^ ]+))`)

// ParseDSN parses a database DSN (data source name) into a ConnConfig
//
// e.g. ParseDSN("user=username password=password host=1.2.3.4 port=5432 dbname=mydb sslmode=disable")
//
// Any options not used by the connection process are parsed into ConnConfig.RuntimeParams.
//
// e.g. ParseDSN("application_name=pgxtest search_path=admin user=username password=password host=1.2.3.4 dbname=mydb")
//
// ParseDSN tries to match libpq behavior with regard to sslmode. See comments
// for ParseEnvLibpq for more information on the security implications of
// sslmode options.
func ParseDSN(s string) (ConnConfig, error) {
	var cp ConnConfig

	m := dsnRegexp.FindAllStringSubmatch(s, -1)

	var sslmode string

	cp.RuntimeParams = make(map[string]string)

	for _, b := range m {
		switch b[1] {
		case "user":
			cp.User = b[2]
		case "password":
			cp.Password = b[2]
		case "host":
			cp.Host = b[2]
		case "port":
			p, err := strconv.ParseUint(b[2], 10, 16)
			if err != nil {
				return cp, err
			}
			cp.Port = uint16(p)
		case "dbname":
			cp.Database = b[2]
		case "sslmode":
			sslmode = b[2]
		default:
			cp.RuntimeParams[b[1]] = b[2]
		}
	}

	err := configSSL(sslmode, &cp)
	if err != nil {
		return cp, err
	}
	if cp.Password == "" {
		pgpass(&cp)
	}
	return cp, nil
}

// ParseConnectionString parses either a URI or a DSN connection string.
// see ParseURI and ParseDSN for details.
func ParseConnectionString(s string) (ConnConfig, error) {
	if strings.HasPrefix(s, "postgres://") || strings.HasPrefix(s, "postgresql://") {
		return ParseURI(s)
	}
	return ParseDSN(s)
}

// ParseEnvLibpq parses the environment like libpq does into a ConnConfig
//
// See http://www.postgresql.org/docs/9.4/static/libpq-envars.html for details
// on the meaning of environment variables.
//
// ParseEnvLibpq currently recognizes the following environment variables:
// PGHOST
// PGPORT
// PGDATABASE
// PGUSER
// PGPASSWORD
// PGSSLMODE
// PGAPPNAME
//
// Important TLS Security Notes:
// ParseEnvLibpq tries to match libpq behavior with regard to PGSSLMODE. This
// includes defaulting to "prefer" behavior if no environment variable is set.
//
// See http://www.postgresql.org/docs/9.4/static/libpq-ssl.html#LIBPQ-SSL-PROTECTION
// for details on what level of security each sslmode provides.
//
// "require" and "verify-ca" modes currently are treated as "verify-full". e.g.
// They have stronger security guarantees than they would with libpq. Do not
// rely on this behavior as it may be possible to match libpq in the future. If
// you need full security use "verify-full".
//
// Several of the PGSSLMODE options (including the default behavior of "prefer")
// will set UseFallbackTLS to true and FallbackTLSConfig to a disabled or
// weakened TLS mode. This means that if ParseEnvLibpq is used, but TLSConfig is
// later set from a different source that UseFallbackTLS MUST be set false to
// avoid the possibility of falling back to weaker or disabled security.
func ParseEnvLibpq() (ConnConfig, error) {
	var cc ConnConfig

	cc.Host = os.Getenv("PGHOST")

	if pgport := os.Getenv("PGPORT"); pgport != "" {
		if port, err := strconv.ParseUint(pgport, 10, 16); err == nil {
			cc.Port = uint16(port)
		} else {
			return cc, err
		}
	}

	cc.Database = os.Getenv("PGDATABASE")
	cc.User = os.Getenv("PGUSER")
	cc.Password = os.Getenv("PGPASSWORD")

	sslmode := os.Getenv("PGSSLMODE")

	err := configSSL(sslmode, &cc)
	if err != nil {
		return cc, err
	}

	cc.RuntimeParams = make(map[string]string)
	if appname := os.Getenv("PGAPPNAME"); appname != "" {
		cc.RuntimeParams["application_name"] = appname
	}
	if cc.Password == "" {
		pgpass(&cc)
	}
	return cc, nil
}

func configSSL(sslmode string, cc *ConnConfig) error {
	// Match libpq default behavior
	if sslmode == "" {
		sslmode = "prefer"
	}

	switch sslmode {
	case "disable":
	case "allow":
		cc.UseFallbackTLS = true
		cc.FallbackTLSConfig = &tls.Config{InsecureSkipVerify: true}
	case "prefer":
		cc.TLSConfig = &tls.Config{InsecureSkipVerify: true}
		cc.UseFallbackTLS = true
		cc.FallbackTLSConfig = nil
	case "require", "verify-ca", "verify-full":
		cc.TLSConfig = &tls.Config{
			ServerName: cc.Host,
		}
	default:
		return errors.New("sslmode is invalid")
	}

	return nil
}

// Prepare creates a prepared statement with name and sql. sql can contain placeholders
// for bound parameters. These placeholders are referenced positional as $1, $2, etc.
//
// Prepare is idempotent; i.e. it is safe to call Prepare multiple times with the same
// name and sql arguments. This allows a code path to Prepare and Query/Exec without
// concern for if the statement has already been prepared.
func (c *Conn) Prepare(name, sql string) (ps *PreparedStatement, err error) {
	return c.PrepareEx(name, sql, nil)
}

// PrepareEx creates a prepared statement with name and sql. sql can contain placeholders
// for bound parameters. These placeholders are referenced positional as $1, $2, etc.
// It defers from Prepare as it allows additional options (such as parameter Oids) to be passed via struct
//
// PrepareEx is idempotent; i.e. it is safe to call PrepareEx multiple times with the same
// name and sql arguments. This allows a code path to PrepareEx and Query/Exec without
// concern for if the statement has already been prepared.
func (c *Conn) PrepareEx(name, sql string, opts *PrepareExOptions) (ps *PreparedStatement, err error) {
	return c.PrepareExContext(context.Background(), name, sql, opts)
}

func (c *Conn) PrepareExContext(ctx context.Context, name, sql string, opts *PrepareExOptions) (ps *PreparedStatement, err error) {
	err = c.waitForPreviousCancelQuery(ctx)
	if err != nil {
		return nil, err
	}

	err = c.initContext(ctx)
	if err != nil {
		return nil, err
	}

	ps, err = c.prepareEx(name, sql, opts)
	err = c.termContext(err)
	return ps, err
}

func (c *Conn) prepareEx(name, sql string, opts *PrepareExOptions) (ps *PreparedStatement, err error) {
	if name != "" {
		if ps, ok := c.preparedStatements[name]; ok && ps.SQL == sql {
			return ps, nil
		}
	}

	if err := c.ensureConnectionReadyForQuery(); err != nil {
		return nil, err
	}

	if c.shouldLog(LogLevelError) {
		defer func() {
			if err != nil {
				c.log(LogLevelError, fmt.Sprintf("Prepare `%s` as `%s` failed: %v", name, sql, err))
			}
		}()
	}

	// parse
	wbuf := newWriteBuf(c, 'P')
	wbuf.WriteCString(name)
	wbuf.WriteCString(sql)

	if opts != nil {
		if len(opts.ParameterOids) > 65535 {
			return nil, fmt.Errorf("Number of PrepareExOptions ParameterOids must be between 0 and 65535, received %d", len(opts.ParameterOids))
		}
		wbuf.WriteInt16(int16(len(opts.ParameterOids)))
		for _, oid := range opts.ParameterOids {
			wbuf.WriteInt32(int32(oid))
		}
	} else {
		wbuf.WriteInt16(0)
	}

	// describe
	wbuf.startMsg('D')
	wbuf.WriteByte('S')
	wbuf.WriteCString(name)

	// sync
	wbuf.startMsg('S')
	wbuf.closeMsg()

	_, err = c.conn.Write(wbuf.buf)
	if err != nil {
		c.die(err)
		return nil, err
	}
	c.readyForQuery = false

	ps = &PreparedStatement{Name: name, SQL: sql}

	var softErr error

	for {
		var t byte
		var r *msgReader
		t, r, err := c.rxMsg()
		if err != nil {
			return nil, err
		}

		switch t {
		case parameterDescription:
			ps.ParameterOids = c.rxParameterDescription(r)

			if len(ps.ParameterOids) > 65535 && softErr == nil {
				softErr = fmt.Errorf("PostgreSQL supports maximum of 65535 parameters, received %d", len(ps.ParameterOids))
			}
		case rowDescription:
			ps.FieldDescriptions = c.rxRowDescription(r)
			for i := range ps.FieldDescriptions {
				if dt, ok := c.ConnInfo.DataTypeForOid(ps.FieldDescriptions[i].DataType); ok {
					ps.FieldDescriptions[i].DataTypeName = dt.Name
					if _, ok := dt.Value.(pgtype.BinaryDecoder); ok {
						ps.FieldDescriptions[i].FormatCode = BinaryFormatCode
					} else {
						ps.FieldDescriptions[i].FormatCode = TextFormatCode
					}
				} else {
					return nil, fmt.Errorf("unknown oid: %d", ps.FieldDescriptions[i].DataType)
				}
			}
		case readyForQuery:
			c.rxReadyForQuery(r)

			if softErr == nil {
				c.preparedStatements[name] = ps
			}

			return ps, softErr
		default:
			if e := c.processContextFreeMsg(t, r); e != nil && softErr == nil {
				softErr = e
			}
		}
	}
}

// Deallocate released a prepared statement
func (c *Conn) Deallocate(name string) error {
	return c.deallocateContext(context.Background(), name)
}

// TODO - consider making this public
func (c *Conn) deallocateContext(ctx context.Context, name string) (err error) {
	err = c.waitForPreviousCancelQuery(ctx)
	if err != nil {
		return err
	}

	err = c.initContext(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = c.termContext(err)
	}()

	if err := c.ensureConnectionReadyForQuery(); err != nil {
		return err
	}

	delete(c.preparedStatements, name)

	// close
	wbuf := newWriteBuf(c, 'C')
	wbuf.WriteByte('S')
	wbuf.WriteCString(name)

	// flush
	wbuf.startMsg('H')
	wbuf.closeMsg()

	_, err = c.conn.Write(wbuf.buf)
	if err != nil {
		c.die(err)
		return err
	}

	for {
		var t byte
		var r *msgReader
		t, r, err := c.rxMsg()
		if err != nil {
			return err
		}

		switch t {
		case closeComplete:
			return nil
		default:
			err = c.processContextFreeMsg(t, r)
			if err != nil {
				return err
			}
		}
	}
}

// Listen establishes a PostgreSQL listen/notify to channel
func (c *Conn) Listen(channel string) error {
	_, err := c.Exec("listen " + quoteIdentifier(channel))
	if err != nil {
		return err
	}

	c.channels[channel] = struct{}{}

	return nil
}

// Unlisten unsubscribes from a listen channel
func (c *Conn) Unlisten(channel string) error {
	_, err := c.Exec("unlisten " + quoteIdentifier(channel))
	if err != nil {
		return err
	}

	delete(c.channels, channel)
	return nil
}

// WaitForNotification waits for a PostgreSQL notification.
func (c *Conn) WaitForNotification(ctx context.Context) (notification *Notification, err error) {
	// Return already received notification immediately
	if len(c.notifications) > 0 {
		notification := c.notifications[0]
		c.notifications = c.notifications[1:]
		return notification, nil
	}

	err = c.waitForPreviousCancelQuery(ctx)
	if err != nil {
		return nil, err
	}

	err = c.initContext(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = c.termContext(err)
	}()

	if err = c.lock(); err != nil {
		return nil, err
	}
	defer func() {
		if unlockErr := c.unlock(); unlockErr != nil && err == nil {
			err = unlockErr
		}
	}()

	if err := c.ensureConnectionReadyForQuery(); err != nil {
		return nil, err
	}

	for {
		t, r, err := c.rxMsg()
		if err != nil {
			return nil, err
		}

		err = c.processContextFreeMsg(t, r)
		if err != nil {
			return nil, err
		}

		if len(c.notifications) > 0 {
			notification := c.notifications[0]
			c.notifications = c.notifications[1:]
			return notification, nil
		}
	}
}

func (c *Conn) IsAlive() bool {
	return atomic.LoadInt32(&c.status) >= connStatusIdle
}

func (c *Conn) CauseOfDeath() error {
	return c.causeOfDeath
}

func (c *Conn) sendQuery(sql string, arguments ...interface{}) (err error) {
	if ps, present := c.preparedStatements[sql]; present {
		return c.sendPreparedQuery(ps, arguments...)
	}
	return c.sendSimpleQuery(sql, arguments...)
}

func (c *Conn) sendSimpleQuery(sql string, args ...interface{}) error {
	if err := c.ensureConnectionReadyForQuery(); err != nil {
		return err
	}

	if len(args) == 0 {
		wbuf := newWriteBuf(c, 'Q')
		wbuf.WriteCString(sql)
		wbuf.closeMsg()

		_, err := c.conn.Write(wbuf.buf)
		if err != nil {
			c.die(err)
			return err
		}
		c.readyForQuery = false

		return nil
	}

	ps, err := c.Prepare("", sql)
	if err != nil {
		return err
	}

	return c.sendPreparedQuery(ps, args...)
}

func (c *Conn) sendPreparedQuery(ps *PreparedStatement, arguments ...interface{}) (err error) {
	if len(ps.ParameterOids) != len(arguments) {
		return fmt.Errorf("Prepared statement \"%v\" requires %d parameters, but %d were provided", ps.Name, len(ps.ParameterOids), len(arguments))
	}

	if err := c.ensureConnectionReadyForQuery(); err != nil {
		return err
	}

	// bind
	wbuf := newWriteBuf(c, 'B')
	wbuf.WriteByte(0)
	wbuf.WriteCString(ps.Name)

	wbuf.WriteInt16(int16(len(ps.ParameterOids)))
	for i, oid := range ps.ParameterOids {
		wbuf.WriteInt16(chooseParameterFormatCode(c.ConnInfo, oid, arguments[i]))
	}

	wbuf.WriteInt16(int16(len(arguments)))
	for i, oid := range ps.ParameterOids {
		if err := encodePreparedStatementArgument(wbuf, oid, arguments[i]); err != nil {
			return err
		}
	}

	wbuf.WriteInt16(int16(len(ps.FieldDescriptions)))
	for _, fd := range ps.FieldDescriptions {
		wbuf.WriteInt16(fd.FormatCode)
	}

	// execute
	wbuf.startMsg('E')
	wbuf.WriteByte(0)
	wbuf.WriteInt32(0)

	// sync
	wbuf.startMsg('S')
	wbuf.closeMsg()

	_, err = c.conn.Write(wbuf.buf)
	if err != nil {
		c.die(err)
	}
	c.readyForQuery = false

	return err
}

// Exec executes sql. sql can be either a prepared statement name or an SQL string.
// arguments should be referenced positionally from the sql string as $1, $2, etc.
func (c *Conn) Exec(sql string, arguments ...interface{}) (commandTag CommandTag, err error) {
	return c.ExecEx(context.Background(), sql, nil, arguments...)
}

// Processes messages that are not exclusive to one context such as
// authentication or query response. The response to these messages is the same
// regardless of when they occur. It also ignores messages that are only
// meaningful in a given context. These messages can occur due to a context
// deadline interrupting message processing. For example, an interrupted query
// may have left DataRow messages on the wire.
func (c *Conn) processContextFreeMsg(t byte, r *msgReader) (err error) {
	switch t {
	case bindComplete:
	case commandComplete:
	case dataRow:
	case emptyQueryResponse:
	case errorResponse:
		return c.rxErrorResponse(r)
	case noData:
	case noticeResponse:
	case notificationResponse:
		c.rxNotificationResponse(r)
	case parameterDescription:
	case parseComplete:
	case readyForQuery:
		c.rxReadyForQuery(r)
	case rowDescription:
	case 'S':
		c.rxParameterStatus(r)

	default:
		return fmt.Errorf("Received unknown message type: %c", t)
	}

	return nil
}

func (c *Conn) rxMsg() (t byte, r *msgReader, err error) {
	if atomic.LoadInt32(&c.status) < connStatusIdle {
		return 0, nil, ErrDeadConn
	}

	t, err = c.mr.rxMsg()
	if err != nil {
		if netErr, ok := err.(net.Error); !(ok && netErr.Timeout()) {
			c.die(err)
		}
	}

	c.lastActivityTime = time.Now()

	if c.shouldLog(LogLevelTrace) {
		c.log(LogLevelTrace, "rxMsg", "type", string(t), "msgBodyLen", len(c.mr.msgBody))
	}

	return t, &c.mr, err
}

func (c *Conn) rxAuthenticationX(r *msgReader) (err error) {
	switch r.readInt32() {
	case 0: // AuthenticationOk
	case 3: // AuthenticationCleartextPassword
		err = c.txPasswordMessage(c.config.Password)
	case 5: // AuthenticationMD5Password
		salt := r.readString(4)
		digestedPassword := "md5" + hexMD5(hexMD5(c.config.Password+c.config.User)+salt)
		err = c.txPasswordMessage(digestedPassword)
	default:
		err = errors.New("Received unknown authentication message")
	}

	return
}

func hexMD5(s string) string {
	hash := md5.New()
	io.WriteString(hash, s)
	return hex.EncodeToString(hash.Sum(nil))
}

func (c *Conn) rxParameterStatus(r *msgReader) {
	key := r.readCString()
	value := r.readCString()
	c.RuntimeParams[key] = value
}

func (c *Conn) rxErrorResponse(r *msgReader) (err PgError) {
	for {
		switch r.readByte() {
		case 'S':
			err.Severity = r.readCString()
		case 'C':
			err.Code = r.readCString()
		case 'M':
			err.Message = r.readCString()
		case 'D':
			err.Detail = r.readCString()
		case 'H':
			err.Hint = r.readCString()
		case 'P':
			s := r.readCString()
			n, _ := strconv.ParseInt(s, 10, 32)
			err.Position = int32(n)
		case 'p':
			s := r.readCString()
			n, _ := strconv.ParseInt(s, 10, 32)
			err.InternalPosition = int32(n)
		case 'q':
			err.InternalQuery = r.readCString()
		case 'W':
			err.Where = r.readCString()
		case 's':
			err.SchemaName = r.readCString()
		case 't':
			err.TableName = r.readCString()
		case 'c':
			err.ColumnName = r.readCString()
		case 'd':
			err.DataTypeName = r.readCString()
		case 'n':
			err.ConstraintName = r.readCString()
		case 'F':
			err.File = r.readCString()
		case 'L':
			s := r.readCString()
			n, _ := strconv.ParseInt(s, 10, 32)
			err.Line = int32(n)
		case 'R':
			err.Routine = r.readCString()

		case 0: // End of error message
			if err.Severity == "FATAL" {
				c.die(err)
			}
			return
		default: // Ignore other error fields
			r.readCString()
		}
	}
}

func (c *Conn) rxBackendKeyData(r *msgReader) {
	c.pid = r.readInt32()
	c.secretKey = r.readInt32()
}

func (c *Conn) rxReadyForQuery(r *msgReader) {
	c.readyForQuery = true
	c.txStatus = r.readByte()
}

func (c *Conn) rxRowDescription(r *msgReader) (fields []FieldDescription) {
	fieldCount := r.readInt16()
	fields = make([]FieldDescription, fieldCount)
	for i := int16(0); i < fieldCount; i++ {
		f := &fields[i]
		f.Name = r.readCString()
		f.Table = pgtype.Oid(r.readUint32())
		f.AttributeNumber = r.readInt16()
		f.DataType = pgtype.Oid(r.readUint32())
		f.DataTypeSize = r.readInt16()
		f.Modifier = r.readInt32()
		f.FormatCode = r.readInt16()
	}
	return
}

func (c *Conn) rxParameterDescription(r *msgReader) (parameters []pgtype.Oid) {
	// Internally, PostgreSQL supports greater than 64k parameters to a prepared
	// statement. But the parameter description uses a 16-bit integer for the
	// count of parameters. If there are more than 64K parameters, this count is
	// wrong. So read the count, ignore it, and compute the proper value from
	// the size of the message.
	r.readInt16()
	parameterCount := len(r.msgBody[r.rp:]) / 4

	parameters = make([]pgtype.Oid, 0, parameterCount)

	for i := 0; i < parameterCount; i++ {
		parameters = append(parameters, pgtype.Oid(r.readUint32()))
	}
	return
}

func (c *Conn) rxNotificationResponse(r *msgReader) {
	n := new(Notification)
	n.PID = r.readInt32()
	n.Channel = r.readCString()
	n.Payload = r.readCString()
	c.notifications = append(c.notifications, n)
}

func (c *Conn) startTLS(tlsConfig *tls.Config) (err error) {
	err = binary.Write(c.conn, binary.BigEndian, []int32{8, 80877103})
	if err != nil {
		return
	}

	response := make([]byte, 1)
	if _, err = io.ReadFull(c.conn, response); err != nil {
		return
	}

	if response[0] != 'S' {
		return ErrTLSRefused
	}

	c.conn = tls.Client(c.conn, tlsConfig)

	return nil
}

func (c *Conn) txStartupMessage(msg *startupMessage) error {
	_, err := c.conn.Write(msg.Bytes())
	return err
}

func (c *Conn) txPasswordMessage(password string) (err error) {
	wbuf := newWriteBuf(c, 'p')
	wbuf.WriteCString(password)
	wbuf.closeMsg()

	_, err = c.conn.Write(wbuf.buf)

	return err
}

func (c *Conn) die(err error) {
	atomic.StoreInt32(&c.status, connStatusClosed)
	c.causeOfDeath = err
	c.conn.Close()
}

func (c *Conn) lock() error {
	if atomic.CompareAndSwapInt32(&c.status, connStatusIdle, connStatusBusy) {
		return nil
	}
	return ErrConnBusy
}

func (c *Conn) unlock() error {
	if atomic.CompareAndSwapInt32(&c.status, connStatusBusy, connStatusIdle) {
		return nil
	}
	return errors.New("unlock conn that is not busy")
}

func (c *Conn) shouldLog(lvl int) bool {
	return c.logger != nil && c.logLevel >= lvl
}

func (c *Conn) log(lvl int, msg string, ctx ...interface{}) {
	if c.pid != 0 {
		ctx = append(ctx, "pid", c.PID)
	}

	c.logger.Log(lvl, msg, ctx...)
}

// SetLogger replaces the current logger and returns the previous logger.
func (c *Conn) SetLogger(logger Logger) Logger {
	oldLogger := c.logger
	c.logger = logger
	return oldLogger
}

// SetLogLevel replaces the current log level and returns the previous log
// level.
func (c *Conn) SetLogLevel(lvl int) (int, error) {
	oldLvl := c.logLevel

	if lvl < LogLevelNone || lvl > LogLevelTrace {
		return oldLvl, ErrInvalidLogLevel
	}

	c.logLevel = lvl
	return lvl, nil
}

func quoteIdentifier(s string) string {
	return `"` + strings.Replace(s, `"`, `""`, -1) + `"`
}

// cancelQuery sends a cancel request to the PostgreSQL server. It returns an
// error if unable to deliver the cancel request, but lack of an error does not
// ensure that the query was canceled. As specified in the documentation, there
// is no way to be sure a query was canceled. See
// https://www.postgresql.org/docs/current/static/protocol-flow.html#AEN112861
func (c *Conn) cancelQuery() {
	if !atomic.CompareAndSwapInt32(&c.cancelQueryInProgress, 0, 1) {
		panic("cancelQuery when cancelQueryInProgress")
	}

	if err := c.conn.SetDeadline(time.Now()); err != nil {
		c.Close() // Close connection if unable to set deadline
		return
	}

	doCancel := func() error {
		network, address := c.config.networkAddress()
		cancelConn, err := c.config.Dial(network, address)
		if err != nil {
			return err
		}
		defer cancelConn.Close()

		// If server doesn't process cancellation request in bounded time then abort.
		err = cancelConn.SetDeadline(time.Now().Add(15 * time.Second))
		if err != nil {
			return err
		}

		buf := make([]byte, 16)
		binary.BigEndian.PutUint32(buf[0:4], 16)
		binary.BigEndian.PutUint32(buf[4:8], 80877102)
		binary.BigEndian.PutUint32(buf[8:12], uint32(c.pid))
		binary.BigEndian.PutUint32(buf[12:16], uint32(c.secretKey))
		_, err = cancelConn.Write(buf)
		if err != nil {
			return err
		}

		_, err = cancelConn.Read(buf)
		if err != io.EOF {
			return fmt.Errorf("Server failed to close connection after cancel query request: %v %v", err, buf)
		}

		return nil
	}

	go func() {
		err := doCancel()
		if err != nil {
			c.Close() // Something is very wrong. Terminate the connection.
		}
		c.cancelQueryCompleted <- struct{}{}
	}()
}

func (c *Conn) Ping() error {
	return c.PingContext(context.Background())
}

func (c *Conn) PingContext(ctx context.Context) error {
	_, err := c.ExecEx(ctx, ";", nil)
	return err
}

func (c *Conn) ExecEx(ctx context.Context, sql string, options *QueryExOptions, arguments ...interface{}) (commandTag CommandTag, err error) {
	err = c.waitForPreviousCancelQuery(ctx)
	if err != nil {
		return "", err
	}

	if err = c.lock(); err != nil {
		return commandTag, err
	}

	startTime := time.Now()
	c.lastActivityTime = startTime

	defer func() {
		if err == nil {
			if c.shouldLog(LogLevelInfo) {
				endTime := time.Now()
				c.log(LogLevelInfo, "Exec", "sql", sql, "args", logQueryArgs(arguments), "time", endTime.Sub(startTime), "commandTag", commandTag)
			}
		} else {
			if c.shouldLog(LogLevelError) {
				c.log(LogLevelError, "Exec", "sql", sql, "args", logQueryArgs(arguments), "error", err)
			}
		}

		if unlockErr := c.unlock(); unlockErr != nil && err == nil {
			err = unlockErr
		}
	}()

	if options != nil && options.SimpleProtocol {
		err = c.initContext(ctx)
		if err != nil {
			return "", err
		}
		defer func() {
			err = c.termContext(err)
		}()

		err = c.sanitizeAndSendSimpleQuery(sql, arguments...)
		if err != nil {
			return "", err

		}
	} else {
		if len(arguments) > 0 {
			ps, ok := c.preparedStatements[sql]
			if !ok {
				var err error
				ps, err = c.PrepareExContext(ctx, "", sql, nil)
				if err != nil {
					return "", err
				}
			}

			err = c.initContext(ctx)
			if err != nil {
				return "", err
			}
			defer func() {
				err = c.termContext(err)
			}()

			err = c.sendPreparedQuery(ps, arguments...)
			if err != nil {
				return "", err
			}
		} else {
			err = c.initContext(ctx)
			if err != nil {
				return "", err
			}
			defer func() {
				err = c.termContext(err)
			}()

			if err = c.sendQuery(sql, arguments...); err != nil {
				return
			}
		}
	}

	var softErr error

	for {
		var t byte
		var r *msgReader
		t, r, err = c.rxMsg()
		if err != nil {
			return commandTag, err
		}

		switch t {
		case readyForQuery:
			c.rxReadyForQuery(r)
			return commandTag, softErr
		case commandComplete:
			commandTag = CommandTag(r.readCString())
		default:
			if e := c.processContextFreeMsg(t, r); e != nil && softErr == nil {
				softErr = e
			}
		}
	}
}

func (c *Conn) initContext(ctx context.Context) error {
	if c.ctxInProgress {
		return errors.New("ctx already in progress")
	}

	if ctx.Done() == nil {
		return nil
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	c.ctxInProgress = true

	go c.contextHandler(ctx)

	return nil
}

func (c *Conn) termContext(opErr error) error {
	if !c.ctxInProgress {
		return opErr
	}

	var err error

	select {
	case err = <-c.closedChan:
		if opErr == nil {
			err = nil
		}
	case c.doneChan <- struct{}{}:
		err = opErr
	}

	c.ctxInProgress = false
	return err
}

func (c *Conn) contextHandler(ctx context.Context) {
	select {
	case <-ctx.Done():
		c.cancelQuery()
		c.closedChan <- ctx.Err()
	case <-c.doneChan:
	}
}

func (c *Conn) waitForPreviousCancelQuery(ctx context.Context) error {
	if atomic.LoadInt32(&c.cancelQueryInProgress) == 0 {
		return nil
	}

	select {
	case <-c.cancelQueryCompleted:
		atomic.StoreInt32(&c.cancelQueryInProgress, 0)
		if err := c.conn.SetDeadline(time.Time{}); err != nil {
			c.Close() // Close connection if unable to disable deadline
			return err
		}
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (c *Conn) ensureConnectionReadyForQuery() error {
	for !c.readyForQuery {
		t, r, err := c.rxMsg()
		if err != nil {
			return err
		}

		switch t {
		case errorResponse:
			pgErr := c.rxErrorResponse(r)
			if pgErr.Severity == "FATAL" {
				return pgErr
			}
		default:
			err = c.processContextFreeMsg(t, r)
			if err != nil {
				return err
			}
		}
	}

	return nil
}
