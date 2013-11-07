package main

import (
	"crypto/rand"
	"fmt"
	"io"
)

type Session struct {
	id     []byte
	userID int32
}

func createSession(userID int32) (id []byte) {
	var err error
	randBytes := make([]byte, 16)
	if _, err = io.ReadFull(rand.Reader, randBytes); err != nil {
		logger.Error(fmt.Sprintf("Unable to create session because unable to read random bytes: %v", err))
		panic("Unable to read random bytes")
	}

	_, err = pool.Execute("insert into sessions(id, user_id) values($1, $2)", randBytes, userID)

	return randBytes
}

func getSession(id []byte) (session Session, present bool) {
	session.id = id
	if userID, err := pool.SelectValue("select user_id from sessions where id=$1", id); err == nil {
		session.userID = userID.(int32)
		present = true
	}

	return
}

func deleteSession(id []byte) {
	pool.Execute("delete from sessions where id=$1", id)
}
