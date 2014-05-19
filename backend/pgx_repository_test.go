package main

import (
	"github.com/JackC/pgx"
	"github.com/vaughan0/go-ini"
	. "gopkg.in/check.v1"
)

var _ = Suite(&RepositorySuite{GetFreshRepository: getFreshPgxRepository})

var sharedPgxRepository *pgxRepository

func getFreshPgxRepository(c *C) repository {
	var err error

	if sharedPgxRepository == nil {
		connPoolConfig := pgx.ConnPoolConfig{MaxConnections: 1, AfterConnect: afterConnect}

		configPath := "../tpr.test.conf"
		file, err := ini.LoadFile(configPath)
		c.Assert(err, IsNil)

		connPoolConfig.ConnConfig, err = extractConnConfig(file)
		c.Assert(err, IsNil)

		sharedPgxRepository, err = NewPgxRepository(connPoolConfig)
		c.Assert(err, IsNil)
	}

	err = sharedPgxRepository.empty()
	c.Assert(err, IsNil)

	return sharedPgxRepository
}
