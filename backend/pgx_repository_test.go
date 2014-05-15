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
		var connectionParameters pgx.ConnectionParameters
		var file ini.File

		configPath := "../tpr.test.conf"
		file, err = ini.LoadFile(configPath)
		c.Assert(err, IsNil)

		connectionParameters, err = extractConnectionOptions(file)
		c.Assert(err, IsNil)

		connectionPoolOptions := pgx.ConnectionPoolOptions{MaxConnections: 1, AfterConnect: afterConnect}
		sharedPgxRepository, err = NewPgxRepository(connectionParameters, connectionPoolOptions)
		c.Assert(err, IsNil)
	}

	err = sharedPgxRepository.empty()
	c.Assert(err, IsNil)

	return sharedPgxRepository
}
