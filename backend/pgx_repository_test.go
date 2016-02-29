package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jackc/tpr/backend/data"
	"net"
	"testing"
	"time"
)

func newUser() *data.User {
	return &data.User{
		Name:           data.NewString("test"),
		PasswordDigest: data.NewBytes([]byte("digest")),
		PasswordSalt:   data.NewBytes([]byte("salt")),
	}
}

func TestPgxRepositoryUsersLifeCycle(t *testing.T) {
	repo := newRepository(t).(*pgxRepository)

	input := &data.User{
		Name:           data.NewString("test"),
		Email:          data.NewString("test@example.com"),
		PasswordDigest: data.NewBytes([]byte("digest")),
		PasswordSalt:   data.NewBytes([]byte("salt")),
	}
	userID, err := repo.CreateUser(input)
	if err != nil {
		t.Fatal(err)
	}

	user, err := repo.GetUserByName(input.Name.Value)
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

	user, err = repo.GetUserByEmail(input.Email.Value)
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

	user, err = data.SelectUserByPK(repo.pool, userID)
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

	u := newUser()
	_, err := repo.CreateUser(u)
	if err != nil {
		t.Fatal(err)
	}

	u = newUser()
	_, err = repo.CreateUser(u)
	if err != (DuplicationError{Field: "name"}) {
		t.Fatalf("Expected %v, got %v", DuplicationError{Field: "name"}, err)
	}
}

func TestPgxRepositoryCreateUserHandlesEmailUniqueness(t *testing.T) {
	repo := newRepository(t)

	u := newUser()
	u.Email = data.NewString("test@example.com")
	_, err := repo.CreateUser(u)
	if err != nil {
		t.Fatal(err)
	}

	u.Name = data.NewString("othername")
	_, err = repo.CreateUser(u)
	if err != (DuplicationError{Field: "email"}) {
		t.Fatalf("Expected %v, got %v", DuplicationError{Field: "email"}, err)
	}
}

func BenchmarkPgxRepositoryGetUser(b *testing.B) {
	repo := newRepository(b).(*pgxRepository)

	userID, err := repo.CreateUser(newUser())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := data.SelectUserByPK(repo.pool, userID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkPgxRepositoryGetUserByName(b *testing.B) {
	repo := newRepository(b)

	user := newUser()
	_, err := repo.CreateUser(user)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetUserByName(user.Name.Value)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func TestPgxRepositoryUpdateUser(t *testing.T) {
	repo := newRepository(t).(*pgxRepository)

	err := repo.UpdateUser(42, &data.User{Name: data.NewString("john")})
	if err != notFound {
		t.Errorf("Expected %#v, got %#v", notFound, err)
	}

	tests := []struct {
		update *data.User
	}{
		{
			update: &data.User{Name: data.NewString("john")},
		},
		{
			update: &data.User{Email: data.NewString("john@example.com")},
		},
		{
			update: &data.User{
				PasswordDigest: data.NewBytes([]byte("newdigest")),
				PasswordSalt:   data.NewBytes([]byte("newsalt")),
			},
		},
		{
			update: &data.User{
				Name:           data.NewString("bill"),
				Email:          data.NewString("bill@example.com"),
				PasswordDigest: data.NewBytes([]byte("newdigest")),
				PasswordSalt:   data.NewBytes([]byte("newsalt")),
			},
		},
	}

	for i, tt := range tests {
		userID, err := repo.CreateUser(&data.User{
			Name:           data.NewString(fmt.Sprintf("test%d", i)),
			Email:          data.NewString(fmt.Sprintf("test%d@example.com", i)),
			PasswordDigest: data.NewBytes([]byte("digest")),
			PasswordSalt:   data.NewBytes([]byte("salt")),
		})
		if err != nil {
			t.Errorf("%d. %v", i, err)
		}

		expected, err := data.SelectUserByPK(repo.pool, userID)
		if err != nil {
			t.Errorf("%d. %v", i, err)
			continue
		}

		if tt.update.ID.Status != data.Undefined {
			expected.ID = tt.update.ID
		}
		if tt.update.Name.Status != data.Undefined {
			expected.Name = tt.update.Name
		}
		if tt.update.Email.Status != data.Undefined {
			expected.Email = tt.update.Email
		}
		if tt.update.PasswordDigest.Status != data.Undefined {
			expected.PasswordDigest = tt.update.PasswordDigest
		}
		if tt.update.PasswordSalt.Status != data.Undefined {
			expected.PasswordSalt = tt.update.PasswordSalt
		}

		err = repo.UpdateUser(userID, tt.update)
		if err != nil {
			t.Errorf("%d. %v", i, err)
			continue
		}

		user, err := data.SelectUserByPK(repo.pool, userID)
		if err != nil {
			t.Errorf("%d. %v", i, err)
		}

		if user.ID != expected.ID {
			t.Errorf("%d. ID was %v, expected %v", i, user.ID, expected.ID)
		}

		if user.Name != expected.Name {
			t.Errorf("%d. Name was %v, expected %v", i, user.Name, expected.Name)
		}

		if user.Email != expected.Email {
			t.Errorf("%d. Email was %v, expected %v", i, user.Email, expected.Email)
		}

		if bytes.Compare(expected.PasswordDigest.Value, user.PasswordDigest.Value) != 0 {
			t.Errorf("%d. PasswordDigest was %v, expected %v", i, user.PasswordDigest, expected.PasswordDigest)
		}

		if bytes.Compare(expected.PasswordSalt.Value, user.PasswordSalt.Value) != 0 {
			t.Errorf("%d. PasswordSalt was %v, expected %v", i, user.PasswordSalt, expected.PasswordSalt)
		}
	}
}

func TestPgxRepositoryFeeds(t *testing.T) {
	repo := newRepository(t)

	userID, err := repo.CreateUser(newUser())
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

	userID, err := repo.CreateUser(newUser())
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

	userID, err := repo.CreateUser(newUser())
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

	userID, err := repo.CreateUser(newUser())
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

	userID, err := repo.CreateUser(newUser())
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

	userID, err := repo.CreateUser(newUser())
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

	userID, err := repo.CreateUser(newUser())
	if err != nil {
		t.Fatal(err)
	}

	sessionID := []byte("deadbeef")

	err = repo.CreateSession(sessionID, userID)
	if err != nil {
		t.Fatal(err)
	}

	user, err := repo.GetUserBySessionID(sessionID)
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

	_, err = repo.GetUserBySessionID(sessionID)
	if err != notFound {
		t.Fatalf("Expected %v, got %v", notFound, err)
	}

	err = repo.DeleteSession(sessionID)
	if err != notFound {
		t.Fatalf("Expected %v, got %v", notFound, err)
	}
}

func TestPgxRepositoryResetPasswordsLifeCycle(t *testing.T) {
	repo := newRepository(t)

	_, localhost, _ := net.ParseCIDR("127.0.0.1/32")
	input := &data.PasswordReset{
		Token:       data.NewString("token"),
		Email:       data.NewString("test@example.com"),
		RequestIP:   data.NewIPNet(*localhost),
		RequestTime: data.NewTime(time.Date(2014, time.May, 30, 16, 10, 0, 0, time.Local)),
	}
	err := repo.CreatePasswordReset(input)
	if err != nil {
		t.Fatal(err)
	}

	reset, err := repo.GetPasswordReset(input.Token.Value)
	if err != nil {
		t.Fatal(err)
	}
	if reset.Token != input.Token {
		t.Errorf("Expected %v, got %v", input.Token, reset.Token)
	}
	if reset.Email != input.Email {
		t.Errorf("Expected %v, got %v", input.Email, reset.Email)
	}
	if reset.RequestIP.Value.String() != input.RequestIP.Value.String() {
		t.Errorf("Expected %v, got %v", input.RequestIP, reset.RequestIP)
	}
	if reset.RequestTime != input.RequestTime {
		t.Errorf("Expected %v, got %v", input.RequestTime, reset.RequestTime)
	}
	if reset.CompletionTime.Status != data.Null {
		t.Errorf("CompletionTime should have been empty, but contained %v", reset.CompletionTime)
	}
	if reset.CompletionIP.Status != data.Null {
		t.Errorf("CompletionIP should have been empty, but contained %v", reset.CompletionIP)
	}

	_, ipnet, _ := net.ParseCIDR("192.168.0.2/32")
	update := &data.PasswordReset{
		CompletionIP:   data.NewIPNet(*ipnet),
		CompletionTime: data.NewTime(time.Date(2014, time.May, 30, 16, 15, 0, 0, time.Local)),
	}

	err = repo.UpdatePasswordReset(input.Token.Value, update)
	if err != nil {
		t.Fatal(err)
	}

	reset, err = repo.GetPasswordReset(input.Token.Value)
	if err != nil {
		t.Fatal(err)
	}
	if reset.Token != input.Token {
		t.Errorf("Expected %v, got %v", input.Token, reset.Token)
	}
	if reset.Email != input.Email {
		t.Errorf("Expected %v, got %v", input.Email, reset.Email)
	}
	if reset.RequestIP.Value.String() != input.RequestIP.Value.String() {
		t.Errorf("Expected %v, got %v", input.RequestIP, reset.RequestIP)
	}
	if reset.RequestTime != input.RequestTime {
		t.Errorf("Expected %v, got %v", input.RequestTime, reset.RequestTime)
	}
	if reset.CompletionIP.Value.String() != update.CompletionIP.Value.String() {
		t.Errorf("Expected %v, got %v", update.CompletionIP, reset.CompletionIP)
	}
	if reset.CompletionTime != update.CompletionTime {
		t.Errorf("Expected %v, got %v", update.CompletionTime, reset.CompletionTime)
	}
}
