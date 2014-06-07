package main

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	log "gopkg.in/inconshreveable/log15.v2"
	"io"
)

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	return nil
}

func genRandPassword() (string, error) {
	return genRandToken(6)
}

func genLostPasswordToken() (string, error) {
	return genRandToken(24)
}

func genRandToken(byteCount int) (string, error) {
	pwBytes := make([]byte, byteCount)
	_, err := rand.Read(pwBytes)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(pwBytes), nil
}

func genSessionID() ([]byte, error) {
	sessionID := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, sessionID)
	if err != nil {
		log.Error("Unable to create session because unable to read random bytes", "error", err)
		return nil, err
	}

	return sessionID, err
}
