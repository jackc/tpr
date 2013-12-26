package main

import (
	"errors"
	"io"
	"time"
)

var notFound = errors.New("not found")

type repository interface {
	createUser(name string, passwordDigest, passwordSalt []byte) (userID int32, err error)
	getUserAuthenticationByName(name string) (userID int32, passwordDigest, passwordSalt []byte, err error)

	createFeed(name, url string) (feedID int32, err error)
	getFeedIDByURL(url string) (feedID int32, err error)
	getFeedsUncheckedSince(since time.Time) (feeds []staleFeed, err error)
	updateFeedWithFetchSuccess(feedID int32, update *parsedFeed, etag string, fetchTime time.Time) error
	updateFeedWithFetchUnchanged(feedID int32, fetchTime time.Time) error
	updateFeedWithFetchFailure(feedID int32, failure string, fetchTime time.Time) (err error)

	copyFeedsAsJSONBySubscribedUserID(w io.Writer, userID int32) error

	createSubscription(userID, feedID int32) (err error)

	createSession(id []byte, userID int32) (err error)
	getUserIDBySessionID(id []byte) (userID int32, err error)
	deleteSession(id []byte) (err error)
}

type staleFeed struct {
	id   int32
	url  string
	etag string
}
