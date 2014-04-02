package main

import (
	"bytes"
	"encoding/json"
	"github.com/JackC/box"
	. "launchpad.net/gocheck"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) }

type RepositorySuite struct {
	GetFreshRepository func(c *C) repository
	repo               repository
}

func (s *RepositorySuite) SetUpTest(c *C) {
	s.repo = s.GetFreshRepository(c)
}

type SubscriptionFromJSON struct {
	ID   int32  `json:id`
	Name string `json:name`
	URL  string `json:url`
}

func (s *RepositorySuite) TestUsers(c *C) {
	name, passwordDigest, passwordSalt := "test", []byte("digest"), []byte("salt")
	userID, err := s.repo.CreateUser(name, passwordDigest, passwordSalt)
	c.Assert(err, IsNil)

	userID2, passwordDigest2, passwordSalt2, err := s.repo.GetUserAuthenticationByName(name)
	c.Assert(err, IsNil)
	c.Check(userID2, Equals, userID)
	c.Check(bytes.Compare(passwordDigest2, passwordDigest), Equals, 0)
	c.Check(bytes.Compare(passwordSalt2, passwordSalt), Equals, 0)

	name2, err := s.repo.GetUserName(userID)
	c.Assert(err, IsNil)
	c.Check(name2, Equals, name)
}

func (s *RepositorySuite) TestFeeds(c *C) {
	userID, err := s.repo.CreateUser("test", []byte("digest"), []byte("salt"))
	c.Assert(err, IsNil)

	now := time.Now()
	fiveMinutesAgo := now.Add(-5 * time.Minute)
	tenMinutesAgo := now.Add(-10 * time.Minute)
	fifteenMinutesAgo := now.Add(-15 * time.Minute)
	update := &parsedFeed{name: "baz", items: make([]parsedItem, 0)}

	// Create a feed
	url := "http://bar"
	err = s.repo.CreateSubscription(userID, url)
	c.Assert(err, IsNil)

	// A new feed has never been fetched -- it should need fetching
	staleFeeds, err := s.repo.GetFeedsUncheckedSince(tenMinutesAgo)
	c.Assert(err, IsNil)
	c.Assert(staleFeeds, HasLen, 1)
	c.Check(staleFeeds[0].URL.MustGet(), Equals, url)

	feedID := staleFeeds[0].ID.MustGet()

	// Update feed as of now
	err = s.repo.UpdateFeedWithFetchSuccess(feedID, update, box.String{}, now)
	c.Assert(err, IsNil)

	// feed should no longer be stale
	staleFeeds, err = s.repo.GetFeedsUncheckedSince(tenMinutesAgo)
	c.Assert(err, IsNil)
	c.Assert(staleFeeds, HasLen, 0)

	// Update feed to be old enough to need refresh
	err = s.repo.UpdateFeedWithFetchSuccess(feedID, update, box.String{}, fifteenMinutesAgo)
	c.Assert(err, IsNil)

	// It should now need fetching
	staleFeeds, err = s.repo.GetFeedsUncheckedSince(tenMinutesAgo)
	c.Assert(err, IsNil)
	c.Assert(staleFeeds, HasLen, 1)
	c.Check(staleFeeds[0].ID.MustGet(), Equals, feedID)

	// But update feed with a recent failed fetch
	err = s.repo.UpdateFeedWithFetchFailure(feedID, "something went wrong", fiveMinutesAgo)
	c.Assert(err, IsNil)

	// feed should no longer be stale
	staleFeeds, err = s.repo.GetFeedsUncheckedSince(tenMinutesAgo)
	c.Assert(err, IsNil)
	c.Assert(staleFeeds, HasLen, 0)
}

func (s *RepositorySuite) TestUpdateFeedWithFetchSuccess(c *C) {
	userID, err := s.repo.CreateUser("test", []byte("digest"), []byte("salt"))
	c.Assert(err, IsNil)

	now := time.Now()

	url := "http://bar"
	err = s.repo.CreateSubscription(userID, url)
	c.Assert(err, IsNil)

	buffer := &bytes.Buffer{}
	err = s.repo.CopySubscriptionsForUserAsJSON(buffer, userID)
	c.Assert(err, IsNil)

	var subscriptions []SubscriptionFromJSON
	err = json.Unmarshal(buffer.Bytes(), &subscriptions)
	c.Assert(err, IsNil)
	feedID := subscriptions[0].ID

	update := &parsedFeed{name: "baz", items: []parsedItem{
		{url: "http://baz/bar", title: "Baz", publicationTime: box.NewTime(now)},
	}}
	err = s.repo.UpdateFeedWithFetchSuccess(feedID, update, box.String{}, now)
	c.Assert(err, IsNil)

	buffer.Reset()
	err = s.repo.CopyUnreadItemsAsJSONByUserID(buffer, userID)
	c.Assert(err, IsNil)

	type UnreadItemsFromJSON struct {
		ID int32 `json:id`
	}

	var unreadItems []UnreadItemsFromJSON
	err = json.Unmarshal(buffer.Bytes(), &unreadItems)
	c.Assert(err, IsNil)
	c.Assert(unreadItems, HasLen, 1)

	// Update again and ensure item does not get created again
	err = s.repo.UpdateFeedWithFetchSuccess(feedID, update, box.String{}, now)
	c.Assert(err, IsNil)

	buffer.Reset()
	err = s.repo.CopyUnreadItemsAsJSONByUserID(buffer, userID)
	c.Assert(err, IsNil)

	err = json.Unmarshal(buffer.Bytes(), &unreadItems)
	c.Assert(err, IsNil)
	c.Assert(unreadItems, HasLen, 1)
}

// This function is a nasty copy and paste of testRepositoryUpdateFeedWithFetchSuccess
// Fix me when refactoring tests
func (s *RepositorySuite) TestUpdateFeedWithFetchSuccessWithoutPublicationTime(c *C) {
	userID, err := s.repo.CreateUser("test", []byte("digest"), []byte("salt"))
	c.Assert(err, IsNil)

	now := time.Now()

	url := "http://bar"
	err = s.repo.CreateSubscription(userID, url)
	c.Assert(err, IsNil)

	buffer := &bytes.Buffer{}
	err = s.repo.CopySubscriptionsForUserAsJSON(buffer, userID)
	c.Assert(err, IsNil)

	var subscriptions []SubscriptionFromJSON
	err = json.Unmarshal(buffer.Bytes(), &subscriptions)
	c.Assert(err, IsNil)
	feedID := subscriptions[0].ID

	update := &parsedFeed{name: "baz", items: []parsedItem{
		{url: "http://baz/bar", title: "Baz"},
	}}
	err = s.repo.UpdateFeedWithFetchSuccess(feedID, update, box.String{}, now)
	c.Assert(err, IsNil)

	buffer.Reset()
	err = s.repo.CopyUnreadItemsAsJSONByUserID(buffer, userID)
	c.Assert(err, IsNil)

	type UnreadItemsFromJSON struct {
		ID int32 `json:id`
	}

	var unreadItems []UnreadItemsFromJSON
	err = json.Unmarshal(buffer.Bytes(), &unreadItems)
	c.Assert(err, IsNil)
	c.Assert(unreadItems, HasLen, 1)

	// Update again and ensure item does not get created again
	err = s.repo.UpdateFeedWithFetchSuccess(feedID, update, box.String{}, now)
	c.Assert(err, IsNil)

	buffer.Reset()
	err = s.repo.CopyUnreadItemsAsJSONByUserID(buffer, userID)
	c.Assert(err, IsNil)

	err = json.Unmarshal(buffer.Bytes(), &unreadItems)
	c.Assert(err, IsNil)
	c.Assert(unreadItems, HasLen, 1)
}

func (s *RepositorySuite) TestSubscriptions(c *C) {
	userID, err := s.repo.CreateUser("test", []byte("digest"), []byte("salt"))
	c.Assert(err, IsNil)

	url := "http://foo"
	err = s.repo.CreateSubscription(userID, url)
	c.Assert(err, IsNil)

	buffer := &bytes.Buffer{}
	err = s.repo.CopySubscriptionsForUserAsJSON(buffer, userID)
	c.Assert(err, IsNil)
	c.Check(bytes.Contains(buffer.Bytes(), []byte("foo")), Equals, true)
}

func (s *RepositorySuite) TestDeleteSubscription(c *C) {
	userID, err := s.repo.CreateUser("test", []byte("digest"), []byte("salt"))
	c.Assert(err, IsNil)

	err = s.repo.CreateSubscription(userID, "http://foo")
	c.Assert(err, IsNil)

	buffer := &bytes.Buffer{}
	err = s.repo.CopySubscriptionsForUserAsJSON(buffer, userID)
	c.Assert(err, IsNil)

	var subscriptions []SubscriptionFromJSON
	err = json.Unmarshal(buffer.Bytes(), &subscriptions)
	c.Assert(err, IsNil)
	feedID := subscriptions[0].ID

	update := &parsedFeed{name: "baz", items: []parsedItem{
		{url: "http://baz/bar", title: "Baz", publicationTime: box.NewTime(time.Now())},
	}}
	err = s.repo.UpdateFeedWithFetchSuccess(feedID, update, box.String{}, time.Now().Add(-20*time.Minute))
	c.Assert(err, IsNil)

	err = s.repo.DeleteSubscription(userID, feedID)
	c.Assert(err, IsNil)

	buffer.Reset()
	err = s.repo.CopySubscriptionsForUserAsJSON(buffer, userID)
	c.Assert(err, IsNil)

	err = json.Unmarshal(buffer.Bytes(), &subscriptions)
	c.Assert(err, IsNil)
	c.Check(subscriptions, HasLen, 0)

	// feed should have been deleted as it was the last user
	staleFeeds, err := s.repo.GetFeedsUncheckedSince(time.Now())
	c.Assert(err, IsNil)
	c.Check(staleFeeds, HasLen, 0)
}

func (s *RepositorySuite) TestCopySubscriptionsForUserAsJSON(c *C) {
	userID, err := s.repo.CreateUser("test", []byte("digest"), []byte("salt"))
	c.Assert(err, IsNil)

	buffer := &bytes.Buffer{}
	err = s.repo.CopySubscriptionsForUserAsJSON(buffer, userID)
	c.Assert(err, IsNil, Commentf("failed when no subscriptions"))

	err = s.repo.CreateSubscription(userID, "http://foo")
	c.Assert(err, IsNil)

	buffer.Reset()
	err = s.repo.CopySubscriptionsForUserAsJSON(buffer, userID)
	c.Assert(err, IsNil)
	c.Check(bytes.Contains(buffer.Bytes(), []byte("foo")), Equals, true)
}

func (s *RepositorySuite) TestSessions(c *C) {
	userID, err := s.repo.CreateUser("test", []byte("digest"), []byte("salt"))
	c.Assert(err, IsNil)

	sessionID := []byte("deadbeef")

	err = s.repo.CreateSession(sessionID, userID)
	c.Assert(err, IsNil)

	userID2, err := s.repo.GetUserIDBySessionID(sessionID)
	c.Assert(err, IsNil)
	c.Check(userID2, Equals, userID)

	err = s.repo.DeleteSession(sessionID)
	c.Assert(err, IsNil)

	_, err = s.repo.GetUserIDBySessionID(sessionID)
	c.Assert(err, Equals, notFound)

	err = s.repo.DeleteSession(sessionID)
	c.Assert(err, Equals, notFound)
}
