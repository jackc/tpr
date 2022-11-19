package backend

import (
	"bytes"
	"crypto/rand"
	"errors"

	"github.com/jackc/tpr/backend/data"
	"golang.org/x/crypto/scrypt"
)

var notFound = errors.New("not found")

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

type staleFeed struct {
	id   int32
	url  string
	etag string
}
