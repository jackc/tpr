package main

import (
	"crypto/rand"
	"encoding/hex"
	"io"
	"sync"
)

type Session struct {
	id     string
	userID int32
}

var sessions map[string]Session
var sessionMutex sync.Mutex

func init() {
	sessions = make(map[string]Session)
}

func createSession(userID int32) (id string) {
	var err error
	randBytes := make([]byte, 16)
	if _, err = io.ReadFull(rand.Reader, randBytes); err != nil {
		panic("Unable to read random bytes")
	}
	id = hex.EncodeToString(randBytes)

	session := Session{id: id, userID: userID}

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
