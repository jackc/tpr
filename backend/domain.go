package main

import (
	"code.google.com/p/go.crypto/scrypt"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"github.com/JackC/box"
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

func CreateUser(name, email, password string) (userID int32, err error) {
	digest, salt, err := digestPassword(password)
	if err != nil {
		return
	}

	user := &User{}
	user.Name.SetCoerceZero(name, box.Empty)
	user.Email.SetCoerceZero(email, box.Empty)
	user.PasswordDigest = digest
	user.PasswordSalt = salt

	return repo.CreateUser(user)
}
