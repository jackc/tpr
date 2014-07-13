package main

import (
	"bytes"
	"code.google.com/p/go.crypto/scrypt"
	"crypto/rand"
	"errors"
	"fmt"
	"github.com/jackc/pgx"
	"io"
	"time"
)

var notFound = errors.New("not found")

type DuplicationError struct {
	Field string // Field or fields that caused the rejection
}

func (e DuplicationError) Error() string {
	return fmt.Sprintf("%s is already taken", e.Field)
}

type repository interface {
	CreateUser(user *User) (userID int32, err error)
	GetUser(userID int32) (*User, error)
	GetUserByName(name string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	UpdateUser(userID int32, attributes *User) error

	CreateSession(id []byte, userID int32) (err error)
	DeleteSession(id []byte) (err error)
	GetUserBySessionID(id []byte) (*User, error)

	CreatePasswordReset(*PasswordReset) error
	GetPasswordReset(token string) (*PasswordReset, error)
	UpdatePasswordReset(string, *PasswordReset) error

	GetFeedsUncheckedSince(since time.Time) (feeds []Feed, err error)
	UpdateFeedWithFetchSuccess(feedID int32, update *parsedFeed, etag pgx.NullString, fetchTime time.Time) error
	UpdateFeedWithFetchUnchanged(feedID int32, fetchTime time.Time) error
	UpdateFeedWithFetchFailure(feedID int32, failure string, fetchTime time.Time) (err error)

	CopyUnreadItemsAsJSONByUserID(w io.Writer, userID int32) error
	CopySubscriptionsForUserAsJSON(w io.Writer, userID int32) error

	MarkItemRead(userID, itemID int32) error

	CreateSubscription(userID int32, feedURL string) (err error)
	GetSubscriptions(userID int32) ([]Subscription, error)
	DeleteSubscription(userID, feedID int32) (err error)
}

type User struct {
	ID             pgx.NullInt32
	Name           pgx.NullString
	Email          pgx.NullString
	PasswordDigest []byte
	PasswordSalt   []byte
}

func (u *User) SetPassword(password string) error {
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

func (u *User) IsPassword(password string) bool {
	digest, err := scrypt.Key([]byte(password), u.PasswordSalt, 16384, 8, 1, 32)
	if err != nil {
		return false
	}

	return bytes.Equal(digest, u.PasswordDigest)
}

type Feed struct {
	ID              pgx.NullInt32
	Name            pgx.NullString
	URL             pgx.NullString
	LastFetchTime   pgx.NullTime
	ETag            pgx.NullString
	LastFailure     pgx.NullString
	LastFailureTime pgx.NullTime
	FailureCount    pgx.NullInt32
	CreationTime    pgx.NullTime
}

type Subscription struct {
	FeedID              pgx.NullInt32
	Name                pgx.NullString
	URL                 pgx.NullString
	LastFetchTime       pgx.NullTime
	LastFailure         pgx.NullString
	LastFailureTime     pgx.NullTime
	FailureCount        pgx.NullInt32
	ItemCount           pgx.NullInt64
	LastPublicationTime pgx.NullTime
}

type PasswordReset struct {
	Token          pgx.NullString
	Email          pgx.NullString
	RequestIP      pgx.NullString
	RequestTime    pgx.NullTime
	UserID         pgx.NullInt32
	CompletionIP   pgx.NullString
	CompletionTime pgx.NullTime
}

type staleFeed struct {
	id   int32
	url  string
	etag string
}
