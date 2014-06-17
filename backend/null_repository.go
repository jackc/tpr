package main

import (
	"errors"
	"github.com/JackC/box"
	"io"
	"time"
)

type nullRepository struct{}

func (repo *nullRepository) CreateUser(user *User) (int32, error) {
	return 0, errors.New("Not implemented")
}

func (repo *nullRepository) GetUser(userID int32) (*User, error) {
	return nil, errors.New("Not implemented")
}

func (repo *nullRepository) GetUserByName(name string) (*User, error) {
	return nil, errors.New("Not implemented")
}

func (repo *nullRepository) UpdateUser(userID int32, attributes *User) error {
	return errors.New("Not implemented")
}

func (repo *nullRepository) UpdateFeedWithFetchSuccess(feedID int32, update *parsedFeed, etag box.String, fetchTime time.Time) (err error) {
	return errors.New("Not implemented")
}

func (repo *nullRepository) UpdateFeedWithFetchUnchanged(feedID int32, fetchTime time.Time) (err error) {
	return errors.New("Not implemented")
}

func (repo *nullRepository) UpdateFeedWithFetchFailure(feedID int32, failure string, fetchTime time.Time) (err error) {
	return errors.New("Not implemented")
}

func (repo *nullRepository) CopySubscriptionsForUserAsJSON(w io.Writer, userID int32) error {
	return errors.New("Not implemented")
}

func (repo *nullRepository) CopyUnreadItemsAsJSONByUserID(w io.Writer, userID int32) error {
	return errors.New("Not implemented")
}

func (repo *nullRepository) MarkItemRead(userID, itemID int32) error {
	return errors.New("Not implemented")
}

func (repo *nullRepository) CreateSubscription(userID int32, feedURL string) error {
	return errors.New("Not implemented")
}

func (repo *nullRepository) GetSubscriptions(userID int32) ([]Subscription, error) {
	return nil, errors.New("Not implemented")
}

func (repo *nullRepository) DeleteSubscription(userID, feedID int32) error {
	return errors.New("Not implemented")
}

func (repo *nullRepository) CreateSession(id []byte, userID int32) (err error) {
	return errors.New("Not implemented")
}

func (repo *nullRepository) GetUserBySessionID(id []byte) (*User, error) {
	return nil, errors.New("Not implemented")
}

func (repo *nullRepository) DeleteSession(id []byte) error {
	return errors.New("Not implemented")
}
