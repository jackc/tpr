package main

import (
	"github.com/vaughan0/go-ini"
	. "gopkg.in/check.v1"
)

var _ = Suite(&RepositorySuite{GetFreshRepository: getFreshPgxRepository})

var sharedPgxRepository *pgxRepository

func getFreshPgxRepository(c *C) repository {
	var err error

	if sharedPgxRepository == nil {
		configPath := "../tpr.test.conf"
		conf, err := ini.LoadFile(configPath)
		if err != nil {
			c.Fatal(err)
		}

		logger, err := newLogger(conf)
		if err != nil {
			c.Fatal(err)
		}

		repo, err := newRepo(conf, logger)
		if err != nil {
			c.Fatal(err)
		}

		sharedPgxRepository = repo.(*pgxRepository)
	}

	err = sharedPgxRepository.empty()
	c.Assert(err, IsNil)

	return sharedPgxRepository
}
