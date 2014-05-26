package main

import (
	. "gopkg.in/check.v1"
)

var _ = Suite(&RepositorySuite{GetFreshRepository: getFreshMemoryRepository})

func getFreshMemoryRepository(c *C) repository {
	repo, err := NewMemoryRepository()
	c.Assert(err, IsNil)
	return repo
}
