package backend

// These tests originally belonged to PgxRepository.

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/tpr/backend/data"
)

func newUser() *data.User {
	return &data.User{
		Name:           pgtype.Text{String: "test", Valid: true},
		PasswordDigest: []byte("digest"),
		PasswordSalt:   []byte("salt"),
	}
}

func TestDataUsersLifeCycle(t *testing.T) {
	pool := newConnPool(t)

	input := &data.User{
		Name:           pgtype.Text{String: "test", Valid: true},
		Email:          pgtype.Text{String: "test@example.com", Valid: true},
		PasswordDigest: []byte("digest"),
		PasswordSalt:   []byte("salt"),
	}
	userID, err := data.CreateUser(context.Background(), pool, input)
	if err != nil {
		t.Fatal(err)
	}

	user, err := data.SelectUserByName(context.Background(), pool, input.Name.String)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID.Int32 != userID {
		t.Errorf("Expected %v, got %v", userID, user.ID)
	}
	if user.Name != input.Name {
		t.Errorf("Expected %v, got %v", input.Name, user.Name)
	}
	if user.Email != input.Email {
		t.Errorf("Expected %v, got %v", input.Email, user.Email)
	}
	if bytes.Compare(user.PasswordDigest, input.PasswordDigest) != 0 {
		t.Errorf("Expected user (%v) and input (%v) PasswordDigest to match, but they did not", user.PasswordDigest, input.PasswordDigest)
	}
	if bytes.Compare(user.PasswordSalt, input.PasswordSalt) != 0 {
		t.Errorf("Expected user (%v), and input (%v) PasswordSalt to match, but they did not", user.PasswordSalt, input.PasswordSalt)
	}

	user, err = data.SelectUserByEmail(context.Background(), pool, input.Email.String)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID.Int32 != userID {
		t.Errorf("Expected %v, got %v", userID, user.ID)
	}
	if user.Name != input.Name {
		t.Errorf("Expected %v, got %v", input.Name, user.Name)
	}
	if user.Email != input.Email {
		t.Errorf("Expected %v, got %v", input.Email, user.Email)
	}
	if bytes.Compare(user.PasswordDigest, input.PasswordDigest) != 0 {
		t.Errorf("Expected user (%v) and input (%v) PasswordDigest to match, but they did not", user.PasswordDigest, input.PasswordDigest)
	}
	if bytes.Compare(user.PasswordSalt, input.PasswordSalt) != 0 {
		t.Errorf("Expected user (%v), and input (%v) PasswordSalt to match, but they did not", user.PasswordSalt, input.PasswordSalt)
	}

	user, err = data.SelectUserByPK(context.Background(), pool, userID)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID.Int32 != userID {
		t.Errorf("Expected %v, got %v", userID, user.ID)
	}
	if user.Name != input.Name {
		t.Errorf("Expected %v, got %v", input.Name, user.Name)
	}
	if user.Email != input.Email {
		t.Errorf("Expected %v, got %v", input.Email, user.Email)
	}
	if bytes.Compare(user.PasswordDigest, input.PasswordDigest) != 0 {
		t.Errorf("Expected user (%v) and input (%v) PasswordDigest to match, but they did not", user.PasswordDigest, input.PasswordDigest)
	}
	if bytes.Compare(user.PasswordSalt, input.PasswordSalt) != 0 {
		t.Errorf("Expected user (%v), and input (%v) PasswordSalt to match, but they did not", user.PasswordSalt, input.PasswordSalt)
	}
}

func TestDataCreateUserHandlesNameUniqueness(t *testing.T) {
	pool := newConnPool(t)

	u := newUser()
	_, err := data.CreateUser(context.Background(), pool, u)
	if err != nil {
		t.Fatal(err)
	}

	u = newUser()
	_, err = data.CreateUser(context.Background(), pool, u)
	if err != (data.DuplicationError{Field: "name"}) {
		t.Fatalf("Expected %v, got %v", data.DuplicationError{Field: "name"}, err)
	}
}

func TestDataCreateUserHandlesEmailUniqueness(t *testing.T) {
	pool := newConnPool(t)

	u := newUser()
	u.Email = pgtype.Text{String: "test@example.com", Valid: true}
	_, err := data.CreateUser(context.Background(), pool, u)
	if err != nil {
		t.Fatal(err)
	}

	u.ID = pgtype.Int4{}
	u.Name = pgtype.Text{String: "othername", Valid: true}
	_, err = data.CreateUser(context.Background(), pool, u)
	if err != (data.DuplicationError{Field: "email"}) {
		t.Fatalf("Expected %v, got %v", data.DuplicationError{Field: "email"}, err)
	}
}

func BenchmarkDataGetUser(b *testing.B) {
	pool := newConnPool(b)

	userID, err := data.CreateUser(context.Background(), pool, newUser())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := data.SelectUserByPK(context.Background(), pool, userID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkDataGetUserByName(b *testing.B) {
	pool := newConnPool(b)

	user := newUser()
	_, err := data.CreateUser(context.Background(), pool, user)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := data.SelectUserByName(context.Background(), pool, user.Name.String)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestDataFeeds(t *testing.T) {
	pool := newConnPool(t)

	userID, err := data.CreateUser(context.Background(), pool, newUser())
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()
	fiveMinutesAgo := now.Add(-5 * time.Minute)
	tenMinutesAgo := now.Add(-10 * time.Minute)
	fifteenMinutesAgo := now.Add(-15 * time.Minute)
	update := &data.ParsedFeed{Name: "baz", Items: make([]data.ParsedItem, 0)}

	// Create a feed
	url := "http://bar"
	err = data.InsertSubscription(context.Background(), pool, userID, url)
	if err != nil {
		t.Fatal(err)
	}

	// A new feed has never been fetched -- it should need fetching
	staleFeeds, err := data.GetFeedsUncheckedSince(context.Background(), pool, tenMinutesAgo)
	if err != nil {
		t.Fatal(err)
	}
	if len(staleFeeds) != 1 {
		t.Fatalf("Found %d stale feed, expected 1", len(staleFeeds))
	}

	if staleFeeds[0].URL != url {
		t.Errorf("Expected %v, got %v", url, staleFeeds[0].URL)
	}

	feedID := staleFeeds[0].ID

	nullString := pgtype.Text{}

	// Update feed as of now
	err = data.UpdateFeedWithFetchSuccess(context.Background(), pool, feedID, update, nullString, now)
	if err != nil {
		t.Fatal(err)
	}

	// feed should no longer be stale
	staleFeeds, err = data.GetFeedsUncheckedSince(context.Background(), pool, tenMinutesAgo)
	if err != nil {
		t.Fatal(err)
	}
	if len(staleFeeds) != 0 {
		t.Fatalf("Found %d stale feed, expected 0", len(staleFeeds))
	}

	// Update feed to be old enough to need refresh
	err = data.UpdateFeedWithFetchSuccess(context.Background(), pool, feedID, update, nullString, fifteenMinutesAgo)
	if err != nil {
		t.Fatal(err)
	}

	// It should now need fetching
	staleFeeds, err = data.GetFeedsUncheckedSince(context.Background(), pool, tenMinutesAgo)
	if err != nil {
		t.Fatal(err)
	}
	if len(staleFeeds) != 1 {
		t.Fatalf("Found %d stale feed, expected 1", len(staleFeeds))
	}
	if staleFeeds[0].ID != feedID {
		t.Errorf("Expected %v, got %v", feedID, staleFeeds[0].ID)
	}

	// But update feed with a recent failed fetch
	err = data.UpdateFeedWithFetchFailure(context.Background(), pool, feedID, "something went wrong", fiveMinutesAgo)
	if err != nil {
		t.Fatal(err)
	}

	// feed should no longer be stale
	staleFeeds, err = data.GetFeedsUncheckedSince(context.Background(), pool, tenMinutesAgo)
	if err != nil {
		t.Fatal(err)
	}
	if len(staleFeeds) != 0 {
		t.Fatalf("Found %d stale feed, expected 0", len(staleFeeds))
	}
}

func TestDataUpdateFeedWithFetchSuccess(t *testing.T) {
	pool := newConnPool(t)

	userID, err := data.CreateUser(context.Background(), pool, newUser())
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	url := "http://bar"
	err = data.InsertSubscription(context.Background(), pool, userID, url)
	if err != nil {
		t.Fatal(err)
	}

	subscriptions, err := data.SelectSubscriptions(context.Background(), pool, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("Found %d subscriptions, expected 1", len(subscriptions))
	}
	feedID := subscriptions[0].FeedID.Int32

	update := &data.ParsedFeed{Name: "baz", Items: []data.ParsedItem{
		{
			URL:             "http://baz/bar",
			Title:           "Baz",
			PublicationTime: pgtype.Timestamptz{Time: now, Valid: true},
		},
	}}

	nullString := pgtype.Text{}

	err = data.UpdateFeedWithFetchSuccess(context.Background(), pool, feedID, update, nullString, now)
	if err != nil {
		t.Fatal(err)
	}

	buffer := &bytes.Buffer{}
	err = data.CopyUnreadItemsAsJSONByUserID(context.Background(), pool, buffer, userID)
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
	err = data.UpdateFeedWithFetchSuccess(context.Background(), pool, feedID, update, nullString, now)
	if err != nil {
		t.Fatal(err)
	}

	buffer.Reset()
	err = data.CopyUnreadItemsAsJSONByUserID(context.Background(), pool, buffer, userID)
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
func TestDataUpdateFeedWithFetchSuccessWithoutPublicationTime(t *testing.T) {
	pool := newConnPool(t)

	userID, err := data.CreateUser(context.Background(), pool, newUser())
	if err != nil {
		t.Fatal(err)
	}

	now := time.Now()

	url := "http://bar"
	err = data.InsertSubscription(context.Background(), pool, userID, url)
	if err != nil {
		t.Fatal(err)
	}

	subscriptions, err := data.SelectSubscriptions(context.Background(), pool, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("Found %d subscriptions, expected 1", len(subscriptions))
	}
	feedID := subscriptions[0].FeedID.Int32

	update := &data.ParsedFeed{Name: "baz", Items: []data.ParsedItem{
		{URL: "http://baz/bar", Title: "Baz"},
	}}

	nullString := pgtype.Text{}

	err = data.UpdateFeedWithFetchSuccess(context.Background(), pool, feedID, update, nullString, now)
	if err != nil {
		t.Fatal(err)
	}

	buffer := &bytes.Buffer{}
	err = data.CopyUnreadItemsAsJSONByUserID(context.Background(), pool, buffer, userID)
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
	err = data.UpdateFeedWithFetchSuccess(context.Background(), pool, feedID, update, nullString, now)
	if err != nil {
		t.Fatal(err)
	}

	buffer.Reset()
	err = data.CopyUnreadItemsAsJSONByUserID(context.Background(), pool, buffer, userID)
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

func TestDataSubscriptions(t *testing.T) {
	pool := newConnPool(t)

	userID, err := data.CreateUser(context.Background(), pool, newUser())
	if err != nil {
		t.Fatal(err)
	}

	url := "http://foo"
	err = data.InsertSubscription(context.Background(), pool, userID, url)
	if err != nil {
		t.Fatal(err)
	}

	subscriptions, err := data.SelectSubscriptions(context.Background(), pool, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("Found %d subscriptions, expected 1", len(subscriptions))
	}
	if subscriptions[0].URL.String != url {
		t.Fatalf("Expected %v, got %v", url, subscriptions[0].URL)
	}
}

func TestDataDeleteSubscription(t *testing.T) {
	pool := newConnPool(t)

	userID, err := data.CreateUser(context.Background(), pool, newUser())
	if err != nil {
		t.Fatal(err)
	}

	err = data.InsertSubscription(context.Background(), pool, userID, "http://foo")
	if err != nil {
		t.Fatal(err)
	}

	subscriptions, err := data.SelectSubscriptions(context.Background(), pool, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 1 {
		t.Fatalf("Found %d subscriptions, expected 1", len(subscriptions))
	}
	feedID := subscriptions[0].FeedID.Int32

	update := &data.ParsedFeed{Name: "baz", Items: []data.ParsedItem{
		{URL: "http://baz/bar",
			Title:           "Baz",
			PublicationTime: pgtype.Timestamptz{Time: time.Now(), Valid: true},
		},
	}}

	nullString := pgtype.Text{}

	err = data.UpdateFeedWithFetchSuccess(context.Background(), pool, feedID, update, nullString, time.Now().Add(-20*time.Minute))
	if err != nil {
		t.Fatal(err)
	}

	err = data.DeleteSubscription(context.Background(), pool, userID, feedID)
	if err != nil {
		t.Fatal(err)
	}

	subscriptions, err = data.SelectSubscriptions(context.Background(), pool, userID)
	if err != nil {
		t.Fatal(err)
	}
	if len(subscriptions) != 0 {
		t.Errorf("Found %d subscriptions, expected 0", len(subscriptions))
	}

	// feed should have been deleted as it was the last user
	staleFeeds, err := data.GetFeedsUncheckedSince(context.Background(), pool, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if len(staleFeeds) != 0 {
		t.Errorf("Found %d staleFeeds, expected 0", len(staleFeeds))
	}
}

func TestDataCopySubscriptionsForUserAsJSON(t *testing.T) {
	pool := newConnPool(t)

	userID, err := data.CreateUser(context.Background(), pool, newUser())
	if err != nil {
		t.Fatal(err)
	}

	buffer := &bytes.Buffer{}
	err = data.CopySubscriptionsForUserAsJSON(context.Background(), pool, buffer, userID)
	if err != nil {
		t.Fatalf("Failed when no subscriptions: %v", err)
	}

	err = data.InsertSubscription(context.Background(), pool, userID, "http://foo")
	if err != nil {
		t.Fatal(err)
	}

	buffer.Reset()
	err = data.CopySubscriptionsForUserAsJSON(context.Background(), pool, buffer, userID)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(buffer.Bytes(), []byte("foo")) != true {
		t.Errorf("Expected %v, got %v", true, bytes.Contains(buffer.Bytes(), []byte("foo")))
	}
}

func TestDataSessions(t *testing.T) {
	pool := newConnPool(t)

	userID, err := data.CreateUser(context.Background(), pool, newUser())
	if err != nil {
		t.Fatal(err)
	}

	sessionID := []byte("deadbeef")

	err = data.InsertSession(context.Background(), pool,
		&data.Session{
			ID:     sessionID,
			UserID: userID,
		},
	)
	if err != nil {
		t.Fatal(err)
	}

	user, err := data.SelectUserBySessionID(context.Background(), pool, sessionID)
	if err != nil {
		t.Fatal(err)
	}
	if user.ID.Int32 != userID {
		t.Errorf("Expected %v, got %v", userID, user.ID)
	}

	err = data.DeleteSession(context.Background(), pool, sessionID)
	if err != nil {
		t.Fatal(err)
	}

	_, err = data.SelectUserBySessionID(context.Background(), pool, sessionID)
	if err != data.ErrNotFound {
		t.Fatalf("Expected %v, got %v", data.ErrNotFound, err)
	}

	err = data.DeleteSession(context.Background(), pool, sessionID)
	if err != data.ErrNotFound {
		t.Fatalf("Expected %v, got %v", notFound, err)
	}
}
