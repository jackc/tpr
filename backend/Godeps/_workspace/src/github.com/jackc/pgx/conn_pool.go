package pgx

import (
	"errors"
	log "gopkg.in/inconshreveable/log15.v2"
	"io"
	"sync"
)

type ConnPoolConfig struct {
	ConnConfig
	MaxConnections int // max simultaneous connections to use, default 5, must be at least 2
	AfterConnect   func(*Conn) error
}

type ConnPool struct {
	allConnections       []*Conn
	availableConnections []*Conn
	cond                 *sync.Cond
	config               ConnConfig // config used when establishing connection
	maxConnections       int
	afterConnect         func(*Conn) error
	logger               log.Logger
}

type ConnPoolStat struct {
	MaxConnections       int // max simultaneous connections to use
	CurrentConnections   int // current live connections
	AvailableConnections int // unused live connections
}

// NewConnPool creates a new ConnPool. config.ConnConfig is passed through to
// Connect directly.
func NewConnPool(config ConnPoolConfig) (p *ConnPool, err error) {
	p = new(ConnPool)
	p.config = config.ConnConfig
	p.maxConnections = config.MaxConnections
	if p.maxConnections == 0 {
		p.maxConnections = 5
	}
	if p.maxConnections < 2 {
		return nil, errors.New("MaxConnections must be at least 2")
	}

	p.afterConnect = config.AfterConnect
	if config.Logger != nil {
		p.logger = config.Logger
	} else {
		p.logger = log.New()
		p.logger.SetHandler(log.DiscardHandler())
	}

	p.allConnections = make([]*Conn, 0, p.maxConnections)
	p.availableConnections = make([]*Conn, 0, p.maxConnections)
	p.cond = sync.NewCond(new(sync.Mutex))

	// Initially establish one connection
	var c *Conn
	c, err = p.createConnection()
	if err != nil {
		return
	}
	p.allConnections = append(p.allConnections, c)
	p.availableConnections = append(p.availableConnections, c)

	return
}

// Acquire takes exclusive use of a connection until it is released.
func (p *ConnPool) Acquire() (c *Conn, err error) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	// A connection is available
	if len(p.availableConnections) > 0 {
		c = p.availableConnections[len(p.availableConnections)-1]
		p.availableConnections = p.availableConnections[:len(p.availableConnections)-1]
		return
	}

	// No connections are available, but we can create more
	if len(p.allConnections) < p.maxConnections {
		c, err = p.createConnection()
		if err != nil {
			return
		}
		p.allConnections = append(p.allConnections, c)
		return
	}

	// All connections are in use and we cannot create more
	if len(p.availableConnections) == 0 {
		p.logger.Warn("All connections in pool are busy - waiting...")
		for len(p.availableConnections) == 0 {
			p.cond.Wait()
		}
	}

	c = p.availableConnections[len(p.availableConnections)-1]
	p.availableConnections = p.availableConnections[:len(p.availableConnections)-1]

	return
}

// Release gives up use of a connection.
func (p *ConnPool) Release(conn *Conn) {
	if conn.TxStatus != 'I' {
		conn.Exec("rollback")
	}

	p.cond.L.Lock()
	if conn.IsAlive() {
		p.availableConnections = append(p.availableConnections, conn)
	} else {
		ac := p.allConnections
		for i, c := range ac {
			if conn == c {
				ac[i] = ac[len(ac)-1]
				p.allConnections = ac[0 : len(ac)-1]
				break
			}
		}
	}
	p.cond.L.Unlock()
	p.cond.Signal()
}

// Close ends the use of a connection by closing all underlying connections.
func (p *ConnPool) Close() {
	for i := 0; i < p.maxConnections; i++ {
		if c, err := p.Acquire(); err != nil {
			_ = c.Close()
		}
	}
}

func (p *ConnPool) Stat() (s ConnPoolStat) {
	p.cond.L.Lock()
	defer p.cond.L.Unlock()

	s.MaxConnections = p.maxConnections
	s.CurrentConnections = len(p.allConnections)
	s.AvailableConnections = len(p.availableConnections)
	return
}

func (p *ConnPool) MaxConnectionCount() int {
	return p.maxConnections
}

func (p *ConnPool) CurrentConnectionCount() int {
	return p.maxConnections
}

func (p *ConnPool) createConnection() (c *Conn, err error) {
	c, err = Connect(p.config)
	if err != nil {
		return
	}
	if p.afterConnect != nil {
		err = p.afterConnect(c)
		if err != nil {
			return
		}
	}
	return
}

// SelectValue acquires a connection, delegates the call to that connection, and releases the connection
func (p *ConnPool) SelectValue(sql string, arguments ...interface{}) (v interface{}, err error) {
	var c *Conn
	if c, err = p.Acquire(); err != nil {
		return
	}
	defer p.Release(c)

	return c.SelectValue(sql, arguments...)
}

// SelectValueTo acquires a connection, delegates the call to that connection, and releases the connection
func (p *ConnPool) SelectValueTo(w io.Writer, sql string, arguments ...interface{}) (err error) {
	var c *Conn
	if c, err = p.Acquire(); err != nil {
		return
	}
	defer p.Release(c)

	return c.SelectValueTo(w, sql, arguments...)
}

// Exec acquires a connection, delegates the call to that connection, and releases the connection
func (p *ConnPool) Exec(sql string, arguments ...interface{}) (commandTag CommandTag, err error) {
	var c *Conn
	if c, err = p.Acquire(); err != nil {
		return
	}
	defer p.Release(c)

	return c.Exec(sql, arguments...)
}

func (p *ConnPool) Query(sql string, args ...interface{}) (*QueryResult, error) {
	c, err := p.Acquire()
	if err != nil {
		// Because checking for errors can be deferred to the *QueryResult, build one with the error
		return &QueryResult{closed: true, err: err}, err
	}

	qr, err := c.Query(sql, args...)
	if err != nil {
		p.Release(c)
		return qr, err
	}

	qr.pool = p
	return qr, nil
}

// Transaction acquires a connection, delegates the call to that connection,
// and releases the connection. The call signature differs slightly from the
// underlying Transaction in that the callback function accepts a *Conn
func (p *ConnPool) Transaction(f func(conn *Conn) bool) (committed bool, err error) {
	var c *Conn
	if c, err = p.Acquire(); err != nil {
		return
	}
	defer p.Release(c)

	return c.Transaction(func() bool {
		return f(c)
	})
}

// TransactionIso acquires a connection, delegates the call to that connection,
// and releases the connection. The call signature differs slightly from the
// underlying TransactionIso in that the callback function accepts a *Conn
func (p *ConnPool) TransactionIso(isoLevel string, f func(conn *Conn) bool) (committed bool, err error) {
	var c *Conn
	if c, err = p.Acquire(); err != nil {
		return
	}
	defer p.Release(c)

	return c.TransactionIso(isoLevel, func() bool {
		return f(c)
	})
}
