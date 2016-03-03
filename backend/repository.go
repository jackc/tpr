package main

import (
	"bytes"
	"crypto/rand"
	"errors"

	"github.com/jackc/tpr/backend/data"
	"golang.org/x/crypto/scrypt"
)

var notFound = errors.New("not found")

type repository interface {
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

	u.PasswordDigest = data.NewBytes(digest)
	u.PasswordSalt = data.NewBytes(salt)

	return nil
}

func IsPassword(u *data.User, password string) bool {
	digest, err := scrypt.Key([]byte(password), u.PasswordSalt.Value, 16384, 8, 1, 32)
	if err != nil {
		return false
	}

	return bytes.Equal(digest, u.PasswordDigest.Value)
}

type staleFeed struct {
	id   int32
	url  string
	etag string
}
