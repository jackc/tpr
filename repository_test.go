package main

import (
	"bytes"
	"encoding/json"
	"github.com/JackC/box"
	"testing"
	"time"
)

type SubscriptionFromJSON struct {
	ID   int32  `json:id`
	Name string `json:name`
	URL  string `json:url`
}

func mustCreateUser(t *testing.T, repo repository, userName string) (userID int32) {
	var err error
	userID, err = repo.CreateUser(userName, []byte("digest"), []byte("salt"))
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	return userID
}

func mustCreateSubscription(t *testing.T, repo repository, userID int32, url string) {
	if err := repo.CreateSubscription(userID, url); err != nil {
		t.Fatalf("CreateSubscription failed: %v", err)
	}
}

func mustUpdateFeedWithFetchSuccess(t *testing.T, repo repository, feedID int32, update *parsedFeed, etag string, fetchTime time.Time) {
	if err := repo.UpdateFeedWithFetchSuccess(feedID, update, etag, fetchTime); err != nil {
		t.Fatalf("CreateSubscription failed: %v", err)
	}
}

func testRepositoryUsers(t *testing.T, repo repository) {
	name, passwordDigest, passwordSalt := "test", []byte("digest"), []byte("salt")
	userID, err := repo.CreateUser(name, passwordDigest, passwordSalt)
	if err != nil {
		t.Fatalf("createUser failed: %v", err)
	}

	userID2, passwordDigest2, passwordSalt2, err := repo.GetUserAuthenticationByName(name)
	if err != nil {
		t.Fatalf("GetUserAuthenticationByName failed: %v", err)
	}
	if userID != userID2 {
		t.Errorf("GetUserAuthenticationByName returned wrong userID: %d instead of %d", userID2, userID)
	}
	if bytes.Compare(passwordDigest, passwordDigest2) != 0 {
		t.Errorf("GetUserAuthenticationByName returned wrong passwordDigest: %v instead of %v", passwordDigest2, passwordDigest)
	}
	if bytes.Compare(passwordSalt, passwordSalt2) != 0 {
		t.Errorf("GetUserAuthenticationByName returned wrong passwordSalt: %v instead of %v", passwordSalt2, passwordSalt)
	}

	name2, err := repo.GetUserName(userID)
	if err != nil {
		t.Fatalf("GetUserName failed: %v", err)
	}
	if name != name2 {
		t.Errorf("GetUserName returned wrong name: %s instead of %s", name2, name)
	}
}

// TODO -- this really needs to be refactored
func testRepositoryFeeds(t *testing.T, repo repository) {
	userID := mustCreateUser(t, repo, "test")
	now := time.Now()
	fiveMinutesAgo := now.Add(-5 * time.Minute)
	tenMinutesAgo := now.Add(-10 * time.Minute)
	fifteenMinutesAgo := now.Add(-15 * time.Minute)
	update := &parsedFeed{name: "baz", items: make([]parsedItem, 0)}

	// Create a feed
	url := "http://bar"
	if err := repo.CreateSubscription(userID, url); err != nil {
		t.Fatalf("CreateSubscription failed: %v", err)
	}

	// A new feed has never been fetched -- it should need fetching
	staleFeeds, err := repo.GetFeedsUncheckedSince(tenMinutesAgo)
	if err != nil {
		t.Fatalf("GetFeedsUncheckedSince failed: %v", err)
	}
	if len(staleFeeds) != 1 {
		t.Fatalf("GetFeedsUncheckedSince returned wrong number of feeds: %d instead of %d", len(staleFeeds), 1)
	}
	if staleFeeds[0].url != url {
		t.Errorf("GetFeedsUncheckedSince returned wrong feed: %s instead of %s", staleFeeds[0].url, url)
	}

	feedID := staleFeeds[0].id

	// Update feed as of now
	err = repo.UpdateFeedWithFetchSuccess(feedID, update, "", now)
	if err != nil {
		t.Fatalf("UpdateFeedWithFetchSuccess failed: %v", err)
	}

	// feed should no longer be stale
	staleFeeds, err = repo.GetFeedsUncheckedSince(tenMinutesAgo)
	if err != nil {
		t.Fatalf("GetFeedsUncheckedSince failed: %v", err)
	}
	if len(staleFeeds) != 0 {
		t.Fatalf("GetFeedsUncheckedSince returned wrong number of feeds: %d instead of %d", len(staleFeeds), 0)
	}

	// Update feed to be old enough to need refresh
	err = repo.UpdateFeedWithFetchSuccess(feedID, update, "", fifteenMinutesAgo)
	if err != nil {
		t.Fatalf("UpdateFeedWithFetchSuccess failed: %v", err)
	}

	// It should now need fetching
	staleFeeds, err = repo.GetFeedsUncheckedSince(tenMinutesAgo)
	if err != nil {
		t.Fatalf("GetFeedsUncheckedSince failed: %v", err)
	}
	if len(staleFeeds) != 1 {
		t.Fatalf("GetFeedsUncheckedSince returned wrong number of feeds: %d instead of %d", len(staleFeeds), 1)
	}
	if staleFeeds[0].id != feedID {
		t.Errorf("GetFeedsUncheckedSince returned wrong feed: %d instead of %d", staleFeeds[0].id, feedID)
	}

	// But update feed with a recent failed fetch
	err = repo.UpdateFeedWithFetchFailure(feedID, "something went wrong", fiveMinutesAgo)
	if err != nil {
		t.Fatalf("UpdateFeedWithFetchSuccess failed: %v", err)
	}

	// feed should no longer be stale
	staleFeeds, err = repo.GetFeedsUncheckedSince(tenMinutesAgo)
	if err != nil {
		t.Fatalf("GetFeedsUncheckedSince failed: %v", err)
	}
	if len(staleFeeds) != 0 {
		t.Fatalf("GetFeedsUncheckedSince returned wrong number of feeds: %d instead of %d", len(staleFeeds), 0)
	}
}

func testRepositoryUpdateFeedWithFetchSuccess(t *testing.T, repo repository) {
	userID := mustCreateUser(t, repo, "test")
	now := time.Now()

	url := "http://bar"
	mustCreateSubscription(t, repo, userID, url)

	buffer := &bytes.Buffer{}
	if err := repo.CopySubscriptionsForUserAsJSON(buffer, userID); err != nil {
		t.Fatalf("CopySubscriptionsForUserAsJSON failed: %v", err)
	}

	var subscriptions []SubscriptionFromJSON
	json.Unmarshal(buffer.Bytes(), &subscriptions)
	feedID := subscriptions[0].ID

	update := &parsedFeed{name: "baz", items: []parsedItem{
		{url: "http://baz/bar", title: "Baz", publicationTime: box.NewTime(now)},
	}}
	err := repo.UpdateFeedWithFetchSuccess(feedID, update, "", now)
	if err != nil {
		t.Fatalf("UpdateFeedWithFetchSuccess failed: %v", err)
	}

	buffer.Reset()
	if err := repo.CopyUnreadItemsAsJSONByUserID(buffer, userID); err != nil {
		t.Fatalf("CopyUnreadItemsAsJSONByUserID failed: %v", err)
	}

	type UnreadItemsFromJSON struct {
		ID int32 `json:id`
	}

	var unreadItems []UnreadItemsFromJSON
	json.Unmarshal(buffer.Bytes(), &unreadItems)
	if len(unreadItems) != 1 {
		t.Fatalf("UpdateFeedWithFetchSuccess should have created %d unread item, but it created %d", 1, len(unreadItems))
	}

	// Update again and ensure item does not get created again
	err = repo.UpdateFeedWithFetchSuccess(feedID, update, "", now)
	if err != nil {
		t.Fatalf("UpdateFeedWithFetchSuccess failed: %v", err)
	}

	buffer.Reset()
	if err := repo.CopyUnreadItemsAsJSONByUserID(buffer, userID); err != nil {
		t.Fatalf("CopyUnreadItemsAsJSONByUserID failed: %v", err)
	}

	json.Unmarshal(buffer.Bytes(), &unreadItems)
	if len(unreadItems) != 1 {
		t.Fatalf("UpdateFeedWithFetchSuccess should have created %d unread item, but it created %d", 1, len(unreadItems))
	}
}

// This function is a nasty copy and paste of testRepositoryUpdateFeedWithFetchSuccess
// Fix me when refactoring tests
func testRepositoryUpdateFeedWithFetchSuccessWithoutPublicationTime(t *testing.T, repo repository) {
	userID := mustCreateUser(t, repo, "test")
	now := time.Now()

	url := "http://bar"
	mustCreateSubscription(t, repo, userID, url)

	buffer := &bytes.Buffer{}
	if err := repo.CopySubscriptionsForUserAsJSON(buffer, userID); err != nil {
		t.Fatalf("CopySubscriptionsForUserAsJSON failed: %v", err)
	}

	var subscriptions []SubscriptionFromJSON
	json.Unmarshal(buffer.Bytes(), &subscriptions)
	feedID := subscriptions[0].ID

	update := &parsedFeed{name: "baz", items: []parsedItem{
		{url: "http://baz/bar", title: "Baz"},
	}}
	err := repo.UpdateFeedWithFetchSuccess(feedID, update, "", now)
	if err != nil {
		t.Fatalf("UpdateFeedWithFetchSuccess failed: %v", err)
	}

	buffer.Reset()
	if err := repo.CopyUnreadItemsAsJSONByUserID(buffer, userID); err != nil {
		t.Fatalf("CopyUnreadItemsAsJSONByUserID failed: %v", err)
	}

	type UnreadItemsFromJSON struct {
		ID int32 `json:id`
	}

	var unreadItems []UnreadItemsFromJSON
	json.Unmarshal(buffer.Bytes(), &unreadItems)
	if len(unreadItems) != 1 {
		t.Fatalf("UpdateFeedWithFetchSuccess should have created %d unread item, but it created %d", 1, len(unreadItems))
	}

	// Update again and ensure item does not get created again
	err = repo.UpdateFeedWithFetchSuccess(feedID, update, "", now)
	if err != nil {
		t.Fatalf("UpdateFeedWithFetchSuccess failed: %v", err)
	}

	buffer.Reset()
	if err := repo.CopyUnreadItemsAsJSONByUserID(buffer, userID); err != nil {
		t.Fatalf("CopyUnreadItemsAsJSONByUserID failed: %v", err)
	}

	json.Unmarshal(buffer.Bytes(), &unreadItems)
	if len(unreadItems) != 1 {
		t.Fatalf("UpdateFeedWithFetchSuccess should have created %d unread item, but it created %d", 1, len(unreadItems))
	}
}

func testRepositorySubscriptions(t *testing.T, repo repository) {
	userID := mustCreateUser(t, repo, "test")
	url := "http://foo"

	if err := repo.CreateSubscription(userID, url); err != nil {
		t.Fatalf("CreateSubscription failed: %v", err)
	}

	buffer := &bytes.Buffer{}
	if err := repo.CopySubscriptionsForUserAsJSON(buffer, userID); err != nil {
		t.Fatalf("CopySubscriptionsForUserAsJSON failed: %v", err)
	}
	if !bytes.Contains(buffer.Bytes(), []byte("foo")) {
		t.Errorf("CopySubscriptionsForUserAsJSON should have included: %v", "foo")
	}
}

func testRepositoryDeleteSubscription(t *testing.T, repo repository) {
	userID := mustCreateUser(t, repo, "test")
	mustCreateSubscription(t, repo, userID, "http://foo")

	buffer := &bytes.Buffer{}
	if err := repo.CopySubscriptionsForUserAsJSON(buffer, userID); err != nil {
		t.Fatalf("CopySubscriptionsForUserAsJSON failed: %v", err)
	}

	var subscriptions []SubscriptionFromJSON
	json.Unmarshal(buffer.Bytes(), &subscriptions)
	feedID := subscriptions[0].ID

	update := &parsedFeed{name: "baz", items: []parsedItem{
		{url: "http://baz/bar", title: "Baz", publicationTime: box.NewTime(time.Now())},
	}}
	mustUpdateFeedWithFetchSuccess(t, repo, feedID, update, "", time.Now().Add(-20*time.Minute))

	if err := repo.DeleteSubscription(userID, feedID); err != nil {
		t.Fatalf("DeleteSubscription failed: %v", err)
	}

	buffer.Reset()
	if err := repo.CopySubscriptionsForUserAsJSON(buffer, userID); err != nil {
		t.Fatalf("CopySubscriptionsForUserAsJSON failed: %v", err)
	}
	json.Unmarshal(buffer.Bytes(), &subscriptions)
	if len(subscriptions) != 0 {
		t.Fatalf("DeleteSubscription did not delete subscription")
	}

	// feed should have been deleted as it was the last user
	staleFeeds, err := repo.GetFeedsUncheckedSince(time.Now())
	if err != nil {
		t.Fatalf("GetFeedsUncheckedSince failed: %v", err)
	}
	if len(staleFeeds) != 0 {
		t.Fatalf("DeleteSubscription did not delete feed when it was last subscription")
	}
}

func testRepositoryCopySubscriptionsForUserAsJSON(t *testing.T, repo repository) {
	userID := mustCreateUser(t, repo, "test")

	buffer := &bytes.Buffer{}
	if err := repo.CopySubscriptionsForUserAsJSON(buffer, userID); err != nil {
		t.Errorf("CopySubscriptionsForUserAsJSON failed when no subscriptions: %v", err)
	}

	mustCreateSubscription(t, repo, userID, "http://foo")
	buffer.Reset()
	if err := repo.CopySubscriptionsForUserAsJSON(buffer, userID); err != nil {
		t.Fatalf("CopySubscriptionsForUserAsJSON failed when one subscription: %v", err)
	}
	if !bytes.Contains(buffer.Bytes(), []byte("foo")) {
		t.Errorf("CopySubscriptionsForUserAsJSON should have included: %v", "foo")
	}
}

func testRepositorySessions(t *testing.T, repo repository) {
	userID, err := repo.CreateUser("test", []byte("digest"), []byte("salt"))
	if err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	sessionID := []byte("deadbeef")

	err = repo.CreateSession(sessionID, userID)
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}

	userID2, err := repo.GetUserIDBySessionID(sessionID)
	if err != nil {
		t.Fatalf("GetUserIDBySessionID failed: %v", err)
	}
	if userID != userID2 {
		t.Errorf("GetUserIDBySessionID returned wrong userID: %d instead of %d", userID2, userID)
	}

	err = repo.DeleteSession(sessionID)
	if err != nil {
		t.Fatalf("DeleteSession failed: %v", err)
	}

	_, err = repo.GetUserIDBySessionID(sessionID)
	if err != notFound {
		t.Fatalf("Should have returned notFound error instead got: %v", err)
	}

	err = repo.DeleteSession(sessionID)
	if err != notFound {
		t.Fatalf("deleteSession should return notFound when deleting non-existent id but it returned: %v", err)
	}
}
