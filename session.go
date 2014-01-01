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
		logger.Error("tpr", fmt.Sprintf("Unable to create session because unable to read random bytes: %v", err))
		panic("Unable to read random bytes")
	}

	err = repo.createSession(randBytes, userID)
	if err != nil {
		logger.Error("tpr", fmt.Sprintf("Unable to create session: %v", err))
	}

	return randBytes
}

func getSession(id []byte) (session Session, present bool) {
	var err error
	session.id = id
	session.userID, err = repo.getUserIDBySessionID(id)
	if err == nil {
		present = true
	}

	return
}

func deleteSession(id []byte) error {
	err := repo.deleteSession(id)
	if err != nil {
		logger.Error("tpr", fmt.Sprintf("Unable to delete session: %v", err))
	}
	return err
}
