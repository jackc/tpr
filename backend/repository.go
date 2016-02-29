package main

import (
	"bytes"
	"crypto/rand"
	"errors"
	"time"

	"github.com/jackc/tpr/backend/data"
	"golang.org/x/crypto/scrypt"
)

var notFound = errors.New("not found")

type repository interface {
	GetFeedsUncheckedSince(since time.Time) (feeds []data.Feed, err error)
	UpdateFeedWithFetchSuccess(feedID int32, update *parsedFeed, etag data.String, fetchTime time.Time) error
	UpdateFeedWithFetchUnchanged(feedID int32, fetchTime time.Time) error
	UpdateFeedWithFetchFailure(feedID int32, failure string, fetchTime time.Time) (err error)
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
