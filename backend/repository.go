package main

import (
	"errors"
	"fmt"
	"github.com/JackC/box"
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
	UpdateUser(userID int32, attributes *User) error

	GetFeedsUncheckedSince(since time.Time) (feeds []Feed, err error)
	UpdateFeedWithFetchSuccess(feedID int32, update *parsedFeed, etag box.String, fetchTime time.Time) error
	UpdateFeedWithFetchUnchanged(feedID int32, fetchTime time.Time) error
	UpdateFeedWithFetchFailure(feedID int32, failure string, fetchTime time.Time) (err error)

	CopyUnreadItemsAsJSONByUserID(w io.Writer, userID int32) error
	CopySubscriptionsForUserAsJSON(w io.Writer, userID int32) error

	MarkItemRead(userID, itemID int32) error

	CreateSubscription(userID int32, feedURL string) (err error)
	GetSubscriptions(userID int32) ([]Subscription, error)
	DeleteSubscription(userID, feedID int32) (err error)

	CreateSession(id []byte, userID int32) (err error)
	GetUserIDBySessionID(id []byte) (userID int32, err error)
	DeleteSession(id []byte) (err error)
}

type User struct {
	ID             box.Int32
	Name           box.String
	Email          box.String
	PasswordDigest []byte
	PasswordSalt   []byte
}

type Feed struct {
	ID              box.Int32
	Name            box.String
	URL             box.String
	LastFetchTime   box.Time
	ETag            box.String
	LastFailure     box.String
	LastFailureTime box.Time
	FailureCount    box.Int32
	CreationTime    box.Time
}

type Subscription struct {
	FeedID              box.Int32
	Name                box.String
	URL                 box.String
	LastFetchTime       box.Time
	LastFailure         box.String
	LastFailureTime     box.Time
	FailureCount        box.Int32
	ItemCount           box.Int64
	LastPublicationTime box.Time
}

type staleFeed struct {
	id   int32
	url  string
	etag string
}
