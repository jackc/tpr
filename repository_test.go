package main

import (
	"bytes"
	"testing"
	"time"
)

func testRepositoryUsers(t *testing.T, repo repository) {
	name, passwordDigest, passwordSalt := "test", []byte("digest"), []byte("salt")
	userID, err := repo.createUser(name, passwordDigest, passwordSalt)
	if err != nil {
		t.Fatalf("createUser failed: %v", err)
	}

	userID2, passwordDigest2, passwordSalt2, err := repo.getUserAuthenticationByName(name)
	if err != nil {
		t.Fatalf("getUserAuthenticationByName failed: %v", err)
	}
	if userID != userID2 {
		t.Errorf("getUserAuthenticationByName returned wrong userID: %d instead of %d", userID2, userID)
	}
	if bytes.Compare(passwordDigest, passwordDigest2) != 0 {
		t.Errorf("getUserAuthenticationByName returned wrong passwordDigest: %v instead of %v", passwordDigest2, passwordDigest)
	}
	if bytes.Compare(passwordSalt, passwordSalt2) != 0 {
		t.Errorf("getUserAuthenticationByName returned wrong passwordSalt: %v instead of %v", passwordSalt2, passwordSalt)
	}
}

// TODO -- this really needs to be refactored
func testRepositoryFeeds(t *testing.T, repo repository) {
	now := time.Now()
	fiveMinutesAgo := now.Add(-5 * time.Minute)
	tenMinutesAgo := now.Add(-10 * time.Minute)
	fifteenMinutesAgo := now.Add(-15 * time.Minute)
	update := &parsedFeed{name: "baz", items: make([]parsedItem, 0)}

	// Create a feed
	name, url := "foo", "http://bar"
	feedID, err := repo.createFeed(name, url)
	if err != nil {
		t.Fatalf("createFeed failed: %v", err)
	}

	// Only way to know if it really worked is if we can retrieve it
	feedID2, err := repo.getFeedIDByURL(url)
	if err != nil {
		t.Fatalf("getFeedIDbyURL failed: %v", err)
	}
	if feedID != feedID2 {
		t.Errorf("getFeedIDbyURL returned wrong feedID: %d instead of %d", feedID2, feedID)
	}

	// A new feed has never been fetched -- it should need fetching
	staleFeeds, err := repo.getFeedsUncheckedSince(tenMinutesAgo)
	if err != nil {
		t.Fatalf("getFeedsUncheckedSince failed: %v", err)
	}
	if len(staleFeeds) != 1 {
		t.Fatalf("getFeedsUncheckedSince returned wrong number of feeds: %d instead of %d", len(staleFeeds), 1)
	}
	if staleFeeds[0].id != feedID {
		t.Errorf("getFeedsUncheckedSince returned wrong feed: %d instead of %d", staleFeeds[0].id, feedID)
	}

	// Update feed as of now
	err = repo.updateFeedWithFetchSuccess(feedID, update, "", now)
	if err != nil {
		t.Fatalf("updateFeedWithFetchSuccess failed: %v", err)
	}

	// feed should no longer be stale
	staleFeeds, err = repo.getFeedsUncheckedSince(tenMinutesAgo)
	if err != nil {
		t.Fatalf("getFeedsUncheckedSince failed: %v", err)
	}
	if len(staleFeeds) != 0 {
		t.Fatalf("getFeedsUncheckedSince returned wrong number of feeds: %d instead of %d", len(staleFeeds), 0)
	}

	// Update feed to be old enough to need refresh
	err = repo.updateFeedWithFetchSuccess(feedID, update, "", fifteenMinutesAgo)
	if err != nil {
		t.Fatalf("updateFeedWithFetchSuccess failed: %v", err)
	}

	// It should now need fetching
	staleFeeds, err = repo.getFeedsUncheckedSince(tenMinutesAgo)
	if err != nil {
		t.Fatalf("getFeedsUncheckedSince failed: %v", err)
	}
	if len(staleFeeds) != 1 {
		t.Fatalf("getFeedsUncheckedSince returned wrong number of feeds: %d instead of %d", len(staleFeeds), 1)
	}
	if staleFeeds[0].id != feedID {
		t.Errorf("getFeedsUncheckedSince returned wrong feed: %d instead of %d", staleFeeds[0].id, feedID)
	}

	// But update feed with a recent failed fetch
	err = repo.updateFeedWithFetchFailure(feedID, "something went wrong", fiveMinutesAgo)
	if err != nil {
		t.Fatalf("updateFeedWithFetchSuccess failed: %v", err)
	}

	// feed should no longer be stale
	staleFeeds, err = repo.getFeedsUncheckedSince(tenMinutesAgo)
	if err != nil {
		t.Fatalf("getFeedsUncheckedSince failed: %v", err)
	}
	if len(staleFeeds) != 0 {
		t.Fatalf("getFeedsUncheckedSince returned wrong number of feeds: %d instead of %d", len(staleFeeds), 0)
	}
}

func testRepositorySubscriptions(t *testing.T, repo repository) {
	userID, err := repo.createUser("test", []byte("digest"), []byte("salt"))
	if err != nil {
		t.Fatalf("createUser failed: %v", err)
	}

	feedID, err := repo.createFeed("foo", "http://bar")
	if err != nil {
		t.Fatalf("createFeed failed: %v", err)
	}

	err = repo.createSubscription(userID, feedID)
	if err != nil {
		t.Fatalf("createSubscription failed: %v", err)
	}

	buffer := &bytes.Buffer{}
	err = repo.copyFeedsAsJSONBySubscribedUserID(buffer, userID)
	if err != nil {
		t.Fatalf("copyFeedsAsJSONBySubscribedUserID failed: %v", err)
	}
	if !bytes.Contains(buffer.Bytes(), []byte("foo")) {
		t.Errorf("copyFeedsAsJSONBySubscribedUserID should have included: %v", "foo")
	}
}

func testRepositorySessions(t *testing.T, repo repository) {
	userID, err := repo.createUser("test", []byte("digest"), []byte("salt"))
	if err != nil {
		t.Fatalf("createUser failed: %v", err)
	}

	sessionID := []byte("deadbeef")

	err = repo.createSession(sessionID, userID)
	if err != nil {
		t.Fatalf("createSession failed: %v", err)
	}

	userID2, err := repo.getUserIDBySessionID(sessionID)
	if err != nil {
		t.Fatalf("getUserIDBySessionID failed: %v", err)
	}
	if userID != userID2 {
		t.Errorf("getUserIDBySessionID returned wrong userID: %d instead of %d", userID2, userID)
	}

	err = repo.deleteSession(sessionID)
	if err != nil {
		t.Fatalf("deleteSession failed: %v", err)
	}

	_, err = repo.getUserIDBySessionID(sessionID)
	if err != notFound {
		t.Fatalf("Should have returned notFound error instead got: %v", err)
	}

	err = repo.deleteSession(sessionID)
	if err != notFound {
		t.Fatalf("deleteSession should return notFound when deleting non-existent id but it returned: %v", err)
	}
}
