package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/jackc/tpr/backend/box"
	"github.com/jackc/tpr/backend/data"
	"testing"
	"time"
)

func newUser() *data.User {
	return &data.User{
		Name:           newString("test"),
		PasswordDigest: data.Bytes{Value: []byte("digest"), Status: data.Present},
		PasswordSalt:   data.Bytes{Value: []byte("salt"), Status: data.Present},
	}
}

func TestPgxRepositoryUsersLifeCycle(t *testing.T) {
	repo := newRepository(t)

	input := &data.User{
		Name:           newString("test"),
		Email:          newString("test@example.com"),
		PasswordDigest: data.Bytes{Value: []byte("digest"), Status: data.Present},
		PasswordSalt:   data.Bytes{Value: []byte("salt"), Status: data.Present},
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

	user, err = repo.GetUser(userID)
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
	u.Email = newString("test@example.com")
	_, err := repo.CreateUser(u)
	if err != nil {
		t.Fatal(err)
	}

	u.Name = newString("othername")
	_, err = repo.CreateUser(u)
	if err != (DuplicationError{Field: "email"}) {
		t.Fatalf("Expected %v, got %v", DuplicationError{Field: "email"}, err)
	}
}

func BenchmarkPgxRepositoryGetUser(b *testing.B) {
	repo := newRepository(b)

	userID, err := repo.CreateUser(newUser())
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := repo.GetUser(userID)
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
	repo := newRepository(t)

	err := repo.UpdateUser(42, &data.User{Name: newString("john")})
	if err != notFound {
		t.Errorf("Expected %#v, got %#v", notFound, err)
	}

	tests := []struct {
		update *data.User
	}{
		{
			update: &data.User{Name: newString("john")},
		},
		{
			update: &data.User{Email: newString("john@example.com")},
		},
		{
			update: &data.User{
				PasswordDigest: data.Bytes{Value: []byte("newdigest"), Status: data.Present},
				PasswordSalt:   data.Bytes{Value: []byte("newsalt"), Status: data.Present},
			},
		},
		{
			update: &data.User{
				Name:           newString("bill"),
				Email:          newString("bill@example.com"),
				PasswordDigest: data.Bytes{Value: []byte("newdigest"), Status: data.Present},
				PasswordSalt:   data.Bytes{Value: []byte("newsalt"), Status: data.Present},
			},
		},
	}

	for i, tt := range tests {
		userID, err := repo.CreateUser(&data.User{
			Name:           newString(fmt.Sprintf("test%d", i)),
			Email:          newString(fmt.Sprintf("test%d@example.com", i)),
			PasswordDigest: data.Bytes{Value: []byte("digest"), Status: data.Present},
			PasswordSalt:   data.Bytes{Value: []byte("salt"), Status: data.Present},
		})
		if err != nil {
			t.Errorf("%d. %v", i, err)
		}

		expected, err := repo.GetUser(userID)
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

		user, err := repo.GetUser(userID)
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
	feedID := subscriptions[0].FeedID.MustGet()

	update := &parsedFeed{name: "baz", items: []parsedItem{
		{url: "http://baz/bar", title: "Baz", publicationTime: box.NewTime(now)},
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
	feedID := subscriptions[0].FeedID.MustGet()

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
	if subscriptions[0].URL.MustGet() != url {
		t.Fatalf("Expected %v, got %v", url, subscriptions[0].URL.MustGet())
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
	feedID := subscriptions[0].FeedID.MustGet()

	update := &parsedFeed{name: "baz", items: []parsedItem{
		{url: "http://baz/bar", title: "Baz", publicationTime: box.NewTime(time.Now())},
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

	input := &PasswordReset{
		Token:       box.NewString("token"),
		Email:       box.NewString("test@example.com"),
		RequestIP:   box.NewString("127.0.0.1/32"),
		RequestTime: box.NewTime(time.Date(2014, time.May, 30, 16, 10, 0, 0, time.Local)),
	}
	err := repo.CreatePasswordReset(input)
	if err != nil {
		t.Fatal(err)
	}

	reset, err := repo.GetPasswordReset(input.Token.MustGet())
	if err != nil {
		t.Fatal(err)
	}
	if reset.Token.GetCoerceNil() != input.Token.GetCoerceNil() {
		t.Errorf("Expected %v, got %v", input.Token.GetCoerceNil(), reset.Token.GetCoerceNil())
	}
	if reset.Email.GetCoerceNil() != input.Email.GetCoerceNil() {
		t.Errorf("Expected %v, got %v", input.Email.GetCoerceNil(), reset.Email.GetCoerceNil())
	}
	if reset.RequestIP.GetCoerceNil() != input.RequestIP.GetCoerceNil() {
		t.Errorf("Expected %v, got %v", input.RequestIP.GetCoerceNil(), reset.RequestIP.GetCoerceNil())
	}
	if reset.RequestTime.GetCoerceNil() != input.RequestTime.GetCoerceNil() {
		t.Errorf("Expected %v, got %v", input.RequestTime.GetCoerceNil(), reset.RequestTime.GetCoerceNil())
	}
	if v, present := reset.CompletionTime.Get(); present {
		t.Errorf("CompletionTime should have been empty, but contained %v", v)
	}
	if v, present := reset.CompletionIP.Get(); present {
		t.Errorf("CompletionIP should have been empty, but contained %v", v)
	}

	update := &PasswordReset{
		CompletionIP:   box.NewString("192.168.0.2/32"),
		CompletionTime: box.NewTime(time.Date(2014, time.May, 30, 16, 15, 0, 0, time.Local)),
	}

	err = repo.UpdatePasswordReset(input.Token.MustGet(), update)
	if err != nil {
		t.Fatal(err)
	}

	reset, err = repo.GetPasswordReset(input.Token.MustGet())
	if err != nil {
		t.Fatal(err)
	}
	if reset.Token.GetCoerceNil() != input.Token.GetCoerceNil() {
		t.Errorf("Expected %v, got %v", input.Token.GetCoerceNil(), reset.Token.GetCoerceNil())
	}
	if reset.Email.GetCoerceNil() != input.Email.GetCoerceNil() {
		t.Errorf("Expected %v, got %v", input.Email.GetCoerceNil(), reset.Email.GetCoerceNil())
	}
	if reset.RequestIP.GetCoerceNil() != input.RequestIP.GetCoerceNil() {
		t.Errorf("Expected %v, got %v", input.RequestIP.GetCoerceNil(), reset.RequestIP.GetCoerceNil())
	}
	if reset.RequestTime.GetCoerceNil() != input.RequestTime.GetCoerceNil() {
		t.Errorf("Expected %v, got %v", input.RequestTime.GetCoerceNil(), reset.RequestTime.GetCoerceNil())
	}
	if reset.CompletionIP.GetCoerceNil() != update.CompletionIP.GetCoerceNil() {
		t.Errorf("Expected %v, got %v", update.CompletionIP.GetCoerceNil(), reset.CompletionIP.GetCoerceNil())
	}
	if reset.CompletionTime.GetCoerceNil() != update.CompletionTime.GetCoerceNil() {
		t.Errorf("Expected %v, got %v", update.CompletionTime.GetCoerceNil(), reset.CompletionTime.GetCoerceNil())
	}
}
