package main

import (
	"github.com/JackC/pgx"
	"github.com/kylelemons/go-gypsy/yaml"
	. "launchpad.net/gocheck"
)

var _ = Suite(&RepositorySuite{GetFreshRepository: getFreshPgxRepository})

var sharedPgxRepository *pgxRepository

func getFreshPgxRepository(c *C) repository {
	var err error

	if sharedPgxRepository == nil {
		var connectionParameters pgx.ConnectionParameters
		var yf *yaml.File

		configPath := "config.test.yml"
		yf, err = yaml.ReadFile(configPath)
		c.Assert(err, IsNil)

		connectionParameters, err = extractConnectionOptions(yf)
		c.Assert(err, IsNil)

		connectionPoolOptions := pgx.ConnectionPoolOptions{MaxConnections: 1, AfterConnect: afterConnect}
		sharedPgxRepository, err = NewPgxRepository(connectionParameters, connectionPoolOptions)
		c.Assert(err, IsNil)
	}

	err = sharedPgxRepository.empty()
	c.Assert(err, IsNil)

	return sharedPgxRepository
}
