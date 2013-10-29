package main

import (
	"encoding/hex"
	"crypto/rand"
	"io"
	"sync"
)

type Session struct {
	id        string
	accountId int32
}

var sessions map[string]Session
var sessionMutex sync.Mutex

func init() {
	sessions = make(map[string]Session)
}

func createSession(accountId int32) (id string) {
	var err error
	randBytes := make([]byte, 16)
	if _, err = io.ReadFull(rand.Reader, randBytes); err != nil {
		panic("Unable to read random bytes")
	}
  id = hex.EncodeToString(randBytes)

	session := Session{id: id, accountId: accountId}

	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	sessions[id] = session
	return id
}

func getSession(id string) (session Session, present bool) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	session, present = sessions[id]
	return
}

func deleteSession(id string) {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()

	delete(sessions, id)
}
