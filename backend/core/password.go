package core

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"

	"log/slog"

	"github.com/jackc/tpr/backend/data"
	"golang.org/x/crypto/scrypt"
)

func validatePassword(password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}

	return nil
}

func GenRandPassword() (string, error) {
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
		slog.Error("Unable to create session because unable to read random bytes", "error", err)
		return nil, err
	}

	return sessionID, err
}

func SetPassword(u *data.User, password string) error {
	salt := make([]byte, 8)
	_, err := rand.Read(salt)
	if err != nil {
		return err
	}

	digest, err := scrypt.Key([]byte(password), salt, 16384, 8, 1, 32)
	if err != nil {
		return err
	}

	u.PasswordDigest = digest
	u.PasswordSalt = salt

	return nil
}

func IsPassword(u *data.User, password string) bool {
	digest, err := scrypt.Key([]byte(password), u.PasswordSalt, 16384, 8, 1, 32)
	if err != nil {
		return false
	}

	return bytes.Equal(digest, u.PasswordDigest)
}
