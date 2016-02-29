package main

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/tpr/backend/data"
)

func newUser() *data.User {
	return &data.User{
		Name:           data.NewString("test"),
		PasswordDigest: data.NewBytes([]byte("digest")),
		PasswordSalt:   data.NewBytes([]byte("salt")),
	}
}

func TestPgxRepositoryUsersLifeCycle(t *testing.T) {
	repo := newRepository(t)
	pool := repo.(*pgxRepository).pool

	input := &data.User{
		Name:           data.NewString("test"),
		Email:          data.NewString("test@example.com"),
		PasswordDigest: data.NewBytes([]byte("digest")),
		PasswordSalt:   data.NewBytes([]byte("salt")),
	}
	userID, err := data.CreateUser(pool, input)
	if err != nil {
		t.Fatal(err)
	}

	user, err := data.SelectUserByName(pool, input.Name.Value)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID.Value != userID {
		t.Errorf("Expected %v, got %v", userID, user.ID)
	}
	if user.Name != input.Name {
		t.Errorf("Expected %v, got %v", input.Name, user.Name)
	}
	if user.Email != input.Email {
		t.Errorf("Expected %v, got %v", input.Email, user.Email)
	}
	if bytes.Compare(user.PasswordDigest.Value, input.PasswordDigest.Value) != 0 {
		t.Errorf("Expected user (%v) and input (%v) PasswordDigest to match, but they did not", user.PasswordDigest, input.PasswordDigest)
	}
	if bytes.Compare(user.PasswordSalt.Value, input.PasswordSalt.Value) != 0 {
		t.Errorf("Expected user (%v), and input (%v) PasswordSalt to match, but they did not", user.PasswordSalt, input.PasswordSalt)
	}

	user, err = data.SelectUserByEmail(pool, input.Email.Value)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID.Value != userID {
		t.Errorf("Expected %v, got %v", userID, user.ID)
	}
	if user.Name != input.Name {
		t.Errorf("Expected %v, got %v", input.Name, user.Name)
	}
	if user.Email != input.Email {
		t.Errorf("Expected %v, got %v", input.Email, user.Email)
	}
	if bytes.Compare(user.PasswordDigest.Value, input.PasswordDigest.Value) != 0 {
		t.Errorf("Expected user (%v) and input (%v) PasswordDigest to match, but they did not", user.PasswordDigest, input.PasswordDigest)
	}
	if bytes.Compare(user.PasswordSalt.Value, input.PasswordSalt.Value) != 0 {
		t.Errorf("Expected user (%v), and input (%v) PasswordSalt to match, but they did not", user.PasswordSalt, input.PasswordSalt)
	}

	user, err = data.SelectUserByPK(pool, userID)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID.Value != userID {
		t.Errorf("Expected %v, got %v", userID, user.ID)
	}
	if user.Name != input.Name {
		t.Errorf("Expected %v, got %v", input.Name, user.Name)
	}
	if user.Email != input.Email {
		t.Errorf("Expected %v, got %v", input.Email, user.Email)
	}
	if bytes.Compare(user.PasswordDigest.Value, input.PasswordDigest.Value) != 0 {
		t.Errorf("Expected user (%v) and input (%v) PasswordDigest to match, but they did not", user.PasswordDigest, input.PasswordDigest)
	}
	if bytes.Compare(user.PasswordSalt.Value, input.PasswordSalt.Value) != 0 {
		t.Errorf("Expected user (%v), and input (%v) PasswordSalt to match, but they did not", user.PasswordSalt, input.PasswordSalt)
	}
}

func TestPgxRepositoryCreateUserHandlesNameUniqueness(t *testing.T) {
	repo := newRepository(t)
	pool := repo.(*pgxRepository).pool

	u := newUser()
	_, err := data.CreateUser(pool, u)
	if err != nil {
		t.Fatal(err)
	}

	u = newUser()
	_, err = data.CreateUser(pool, u)
	if err != (data.DuplicationError{Field: "name"}) {
		t.Fatalf("Expected %v, got %v", data.DuplicationError{Field: "name"}, err)
	}
}

func TestPgxRepositoryCreateUserHandlesEmailUniqueness(t *testing.T) {
	repo := newRepository(t)
	pool := repo.(*pgxRepository).pool

	u := newUser()
	u.Email = data.NewString("test@example.com")
	_, err := data.CreateUser(pool, u)
	if err != nil {
		t.Fatal(err)
	}

	u.Name = data.NewString("othername")
	_, err = data.CreateUser(pool, u)
	if err != (data.DuplicationError{Field: "email"}) {
		t.Fatalf("Expected %v, got %v", data.DuplicationError{Field: "email"}, err)
	}
}

func BenchmarkPgxRepositoryGetUser(b *testing.B) {
	repo := newRepository(b)
	pool := repo.(*pgxRepository).pool

	userID, err := data.CreateUser(pool, newUser())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := data.SelectUserByPK(pool, userID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPgxRepositoryGetUserByName(b *testing.B) {
	repo := newRepository(b)
	pool := repo.(*pgxRepository).pool

	user := newUser()
	_, err := data.CreateUser(pool, user)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := data.SelectUserByName(pool, user.Name.Value)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestPgxRepositoryFeeds(t *testing.T) {
	repo := newRepository(t)
	pool := repo.(*pgxRepository).pool

	userID, err := data.CreateUser(pool, newUser())
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	fiveMinutesAgo := now.Add(-5 * time.Minute)
	tenMinutesAgo := now.Add(-10 * time.Minute)
	fifteenMinutesAgo := now.Add(-15 * time.Minute)
	update := &parsedFeed{name: "baz", items: make([]parsedItem, 0)}

	// Create a feed
	url := "http://bar"
	err = repo.CreateSubscription(userID, url)
	if err != nil {
		t.Fatal(err)
	}

	// A new feed has never been fetched -- it should need fetching
	staleFeeds, err := repo.GetFeedsUncheckedSince(tenMinutesAgo)
	if err != nil {
		t.Fatal(err)
	}
	if len(staleFeeds) != 1 {
		t.Fatalf("Found %d stale feed, expected 1", len(staleFeeds))
	}

	if staleFeeds[0].URL.Value != url {
		t.Errorf("Expected %v, got %v", url, staleFeeds[0].URL)
	}

	feedID := staleFeeds[0].ID.Value

	nullString := data.String{Status: data.Null}

	// Update feed as of now
	err = repo.UpdateFeedWithFetchSuccess(feedID, update, nullString, now)
	if err != nil {
		t.Fatal(err)
	}

	// feed should no longer be stale
	staleFeeds, err = repo.GetFeedsUncheckedSince(tenMinutesAgo)
	if err != nil {
		t.Fatal(err)
	}
	if len(staleFeeds) != 0 {
		t.Fatalf("Found %d stale feed, expected 0", len(staleFeeds))
	}

	// Update feed to be old enough to need refresh
	err = repo.UpdateFeedWithFetchSuccess(feedID, update, nullString, fifteenMinutesAgo)
	if err != nil {
		t.Fatal(err)
	}

	// It should now need fetching
	staleFeeds, err = repo.GetFeedsUncheckedSince(tenMinutesAgo)
	if err != nil {
		t.Fatal(err)
	}
	if len(staleFeeds) != 1 {
		t.Fatalf("Found %d stale feed, expected 1", len(staleFeeds))
	}
	if staleFeeds[0].ID.Value != feedID {
		t.Errorf("Expected %v, got %v", feedID, staleFeeds[0].ID)
	}

	// But update feed with a recent failed fetch
	err = repo.UpdateFeedWithFetchFailure(feedID, "something went wrong", fiveMinutesAgo)
	if err != nil {
		t.Fatal(err)
	}

	// feed should no longer be stale
	staleFeeds, err = repo.GetFeedsUncheckedSince(tenMinutesAgo)
	if err != nil {
		t.Fatal(err)
	}
	if len(staleFeeds) != 0 {
		t.Fatalf("Found %d stale feed, expected 0", len(staleFeeds))
	}
}

func TestPgxRepositoryUpdateFeedWithFetchSuccess(t *testing.T) {
	repo := newRepository(t)
	pool := repo.(*pgxRepository).pool

	userID, err := data.CreateUser(pool, newUser())
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	url := "http://bar"
	err = repo.CreateSubscription(userID, url)
	if err != nil {
		t.Fatal(err)
	}

	subscriptions, err := repo.GetSubscriptions(userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("Found %d subscriptions, expected 1", len(subscriptions))
	}
	feedID := subscriptions[0].FeedID.Value

	update := &parsedFeed{name: "baz", items: []parsedItem{
		{url: "http://baz/bar", title: "Baz", publicationTime: data.NewTime(now)},
	}}

	nullString := data.String{Status: data.Null}

	err = repo.UpdateFeedWithFetchSuccess(feedID, update, nullString, now)
	if err != nil {
		t.Fatal(err)
	}

	buffer := &bytes.Buffer{}
	err = repo.CopyUnreadItemsAsJSONByUserID(buffer, userID)
	if err != nil {
		t.Fatal(err)
	}

	type UnreadItemsFromJSON struct {
		ID int32 `json:id`
	}

	var unreadItems []UnreadItemsFromJSON
	err = json.Unmarshal(buffer.Bytes(), &unreadItems)
	if err != nil {
		t.Fatal(err)
	}
	if len(unreadItems) != 1 {
		t.Fatalf("Found %d unreadItems, expected 1", len(unreadItems))
	}

	// Update again and ensure item does not get created again
	err = repo.UpdateFeedWithFetchSuccess(feedID, update, nullString, now)
	if err != nil {
		t.Fatal(err)
	}

	buffer.Reset()
	err = repo.CopyUnreadItemsAsJSONByUserID(buffer, userID)
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(buffer.Bytes(), &unreadItems)
	if err != nil {
		t.Fatal(err)
	}
	if len(unreadItems) != 1 {
		t.Fatalf("Found %d unreadItems, expected 1", len(unreadItems))
	}
}

// This function is a nasty copy and paste of testRepositoryUpdateFeedWithFetchSuccess
// Fix me when refactoring tests
func TestPgxRepositoryUpdateFeedWithFetchSuccessWithoutPublicationTime(t *testing.T) {
	repo := newRepository(t)
	pool := repo.(*pgxRepository).pool

	userID, err := data.CreateUser(pool, newUser())
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	url := "http://bar"
	err = repo.CreateSubscription(userID, url)
	if err != nil {
		t.Fatal(err)
	}

	subscriptions, err := repo.GetSubscriptions(userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("Found %d subscriptions, expected 1", len(subscriptions))
	}
	feedID := subscriptions[0].FeedID.Value

	update := &parsedFeed{name: "baz", items: []parsedItem{
		{url: "http://baz/bar", title: "Baz"},
	}}

	nullString := data.String{Status: data.Null}

	err = repo.UpdateFeedWithFetchSuccess(feedID, update, nullString, now)
	if err != nil {
		t.Fatal(err)
	}

	buffer := &bytes.Buffer{}
	err = repo.CopyUnreadItemsAsJSONByUserID(buffer, userID)
	if err != nil {
		t.Fatal(err)
	}

	type UnreadItemsFromJSON struct {
		ID int32 `json:id`
	}

	var unreadItems []UnreadItemsFromJSON
	err = json.Unmarshal(buffer.Bytes(), &unreadItems)
	if err != nil {
		t.Fatal(err)
	}
	if len(unreadItems) != 1 {
		t.Fatalf("Found %d unreadItems, expected 1", len(unreadItems))
	}

	// Update again and ensure item does not get created again
	err = repo.UpdateFeedWithFetchSuccess(feedID, update, nullString, now)
	if err != nil {
		t.Fatal(err)
	}

	buffer.Reset()
	err = repo.CopyUnreadItemsAsJSONByUserID(buffer, userID)
	if err != nil {
		t.Fatal(err)
	}

	err = json.Unmarshal(buffer.Bytes(), &unreadItems)
	if err != nil {
		t.Fatal(err)
	}
	if len(unreadItems) != 1 {
		t.Fatalf("Found %d unreadItems, expected 1", len(unreadItems))
	}
}

func TestPgxRepositorySubscriptions(t *testing.T) {
	repo := newRepository(t)
	pool := repo.(*pgxRepository).pool

	userID, err := data.CreateUser(pool, newUser())
	if err != nil {
		t.Fatal(err)
	}

	url := "http://foo"
	err = repo.CreateSubscription(userID, url)
	if err != nil {
		t.Fatal(err)
	}

	subscriptions, err := repo.GetSubscriptions(userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("Found %d subscriptions, expected 1", len(subscriptions))
	}
	if subscriptions[0].URL.Value != url {
		t.Fatalf("Expected %v, got %v", url, subscriptions[0].URL)
	}
}

func TestPgxRepositoryDeleteSubscription(t *testing.T) {
	repo := newRepository(t)
	pool := repo.(*pgxRepository).pool

	userID, err := data.CreateUser(pool, newUser())
	if err != nil {
		t.Fatal(err)
	}

	err = repo.CreateSubscription(userID, "http://foo")
	if err != nil {
		t.Fatal(err)
	}

	subscriptions, err := repo.GetSubscriptions(userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("Found %d subscriptions, expected 1", len(subscriptions))
	}
	feedID := subscriptions[0].FeedID.Value

	update := &parsedFeed{name: "baz", items: []parsedItem{
		{url: "http://baz/bar", title: "Baz", publicationTime: data.NewTime(time.Now())},
	}}

	nullString := data.String{Status: data.Null}

	err = repo.UpdateFeedWithFetchSuccess(feedID, update, nullString, time.Now().Add(-20*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	err = repo.DeleteSubscription(userID, feedID)
	if err != nil {
		t.Fatal(err)
	}

	subscriptions, err = repo.GetSubscriptions(userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 0 {
		t.Errorf("Found %d subscriptions, expected 0", len(subscriptions))
	}

	// feed should have been deleted as it was the last user
	staleFeeds, err := repo.GetFeedsUncheckedSince(time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(staleFeeds) != 0 {
		t.Errorf("Found %d staleFeeds, expected 0", len(staleFeeds))
	}
}

func TestPgxRepositoryCopySubscriptionsForUserAsJSON(t *testing.T) {
	repo := newRepository(t)
	pool := repo.(*pgxRepository).pool

	userID, err := data.CreateUser(pool, newUser())
	if err != nil {
		t.Fatal(err)
	}

	buffer := &bytes.Buffer{}
	err = repo.CopySubscriptionsForUserAsJSON(buffer, userID)
	if err != nil {
		t.Fatalf("Failed when no subscriptions: %v", err)
	}

	err = repo.CreateSubscription(userID, "http://foo")
	if err != nil {
		t.Fatal(err)
	}

	buffer.Reset()
	err = repo.CopySubscriptionsForUserAsJSON(buffer, userID)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(buffer.Bytes(), []byte("foo")) != true {
		t.Errorf("Expected %v, got %v", true, bytes.Contains(buffer.Bytes(), []byte("foo")))
	}
}

func TestPgxRepositorySessions(t *testing.T) {
	repo := newRepository(t)
	pool := repo.(*pgxRepository).pool

	userID, err := data.CreateUser(pool, newUser())
	if err != nil {
		t.Fatal(err)
	}

	sessionID := []byte("deadbeef")

	err = repo.CreateSession(sessionID, userID)
	if err != nil {
		t.Fatal(err)
	}

	user, err := data.SelectUserBySessionID(pool, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID.Value != userID {
		t.Errorf("Expected %v, got %v", userID, user.ID)
	}

	err = repo.DeleteSession(sessionID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = data.SelectUserBySessionID(pool, sessionID)
	if err != data.ErrNotFound {
		t.Fatalf("Expected %v, got %v", data.ErrNotFound, err)
	}

	err = repo.DeleteSession(sessionID)
	if err != notFound {
		t.Fatalf("Expected %v, got %v", notFound, err)
	}
}
