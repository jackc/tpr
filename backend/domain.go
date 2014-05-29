package main

import (
	"code.google.com/p/go.crypto/scrypt"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
)

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	return nil
}

func genRandPassword() (string, error) {
	pwBytes := make([]byte, 6)
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
		logger.Error("tpr", fmt.Sprintf("Unable to create session because unable to read random bytes: %v", err))
		return nil, err
	}

	return sessionID, err
}

func digestPassword(password string) ([]byte, []byte, error) {
	salt := make([]byte, 8)
	_, err := rand.Read(salt)
	if err != nil {
		return nil, nil, err
	}

	digest, err := scrypt.Key([]byte(password), salt, 16384, 8, 1, 32)
	if err != nil {
		return nil, nil, err
	}

	return digest, salt, nil
}
