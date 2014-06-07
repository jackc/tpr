package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/JackC/box"
	. "gopkg.in/check.v1"
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

func (s *RepositorySuite) newUser() *User {
	return &User{
		Name:           box.NewString("test"),
		PasswordDigest: []byte("digest"),
		PasswordSalt:   []byte("salt"),
	}
}

func (s *RepositorySuite) TestUsersLifeCycle(c *C) {
	input := &User{
		Name:           box.NewString("test"),
		Email:          box.NewString("test@example.com"),
		PasswordDigest: []byte("digest"),
		PasswordSalt:   []byte("salt"),
	}
	userID, err := s.repo.CreateUser(input)
	c.Assert(err, IsNil)

	user, err := s.repo.GetUserByName(input.Name.MustGet())
	c.Assert(err, IsNil)
	c.Check(user.ID.GetCoerceNil(), Equals, userID)
	c.Check(user.Name.GetCoerceNil(), Equals, input.Name.MustGet())
	c.Check(user.Email.GetCoerceNil(), Equals, input.Email.MustGet())
	c.Check(bytes.Compare(user.PasswordDigest, input.PasswordDigest), Equals, 0)
	c.Check(bytes.Compare(user.PasswordSalt, input.PasswordSalt), Equals, 0)

	user, err = s.repo.GetUserByEmail(input.Email.MustGet())
	c.Assert(err, IsNil)
	c.Check(user.ID.GetCoerceNil(), Equals, userID)
	c.Check(user.Name.GetCoerceNil(), Equals, input.Name.MustGet())
	c.Check(user.Email.GetCoerceNil(), Equals, input.Email.MustGet())
	c.Check(bytes.Compare(user.PasswordDigest, input.PasswordDigest), Equals, 0)
	c.Check(bytes.Compare(user.PasswordSalt, input.PasswordSalt), Equals, 0)

	user, err = s.repo.GetUser(userID)
	c.Assert(err, IsNil)
	c.Check(user.ID.GetCoerceNil(), Equals, userID)
	c.Check(user.Name.GetCoerceNil(), Equals, input.Name.MustGet())
	c.Check(user.Email.GetCoerceNil(), Equals, input.Email.MustGet())
	c.Check(bytes.Compare(user.PasswordDigest, input.PasswordDigest), Equals, 0)
	c.Check(bytes.Compare(user.PasswordSalt, input.PasswordSalt), Equals, 0)
}

func (s *RepositorySuite) TestCreateUserHandlesNameUniqueness(c *C) {
	u := s.newUser()
	_, err := s.repo.CreateUser(u)
	c.Assert(err, IsNil)

	_, err = s.repo.CreateUser(u)
	c.Assert(err, Equals, DuplicationError{Field: "name"})
}

func (s *RepositorySuite) TestCreateUserHandlesEmailUniqueness(c *C) {
	u := s.newUser()
	u.Email.Set("test@example.com")
	_, err := s.repo.CreateUser(u)
	c.Assert(err, IsNil)

	u.Name.Set("othername")
	_, err = s.repo.CreateUser(u)
	c.Assert(err, Equals, DuplicationError{Field: "email"})
}

func (s *RepositorySuite) BenchmarkGetUser(c *C) {
	userID, err := s.repo.CreateUser(s.newUser())
	c.Assert(err, IsNil)

	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		_, err := s.repo.GetUser(userID)
		c.Assert(err, IsNil)
	}
}

func (s *RepositorySuite) BenchmarkGetUserByName(c *C) {
	user := s.newUser()
	_, err := s.repo.CreateUser(user)
	c.Assert(err, IsNil)

	c.ResetTimer()
	for i := 0; i < c.N; i++ {
		_, err := s.repo.GetUserByName(user.Name.MustGet())
		c.Assert(err, IsNil)
	}
}

func (s *RepositorySuite) TestUpdateUser(c *C) {
	err := s.repo.UpdateUser(42, &User{Name: box.NewString("john")})
	c.Check(err, Equals, notFound)

	tests := []struct {
		update *User
	}{
		{
			update: &User{Name: box.NewString("john")},
		},
		{
			update: &User{Email: box.NewString("john@example.com")},
		},
		{
			update: &User{
				PasswordDigest: []byte("newdigest"),
				PasswordSalt:   []byte("newsalt"),
			},
		},
		{
			update: &User{
				Name:           box.NewString("bill"),
				Email:          box.NewString("bill@example.com"),
				PasswordDigest: []byte("newdigest"),
				PasswordSalt:   []byte("newsalt"),
			},
		},
	}

	for i, t := range tests {
		userID, err := s.repo.CreateUser(&User{
			Name:           box.NewString(fmt.Sprintf("test%d", i)),
			Email:          box.NewString(fmt.Sprintf("test%d@example.com", i)),
			PasswordDigest: []byte("digest"),
			PasswordSalt:   []byte("salt"),
		})
		c.Check(err, IsNil, Commentf("%d", i))

		err = s.repo.UpdateUser(userID, t.update)
		c.Check(err, IsNil, Commentf("%d", i))

		user, err := s.repo.GetUser(userID)
		c.Check(err, IsNil, Commentf("%d", i))

		if id, ok := t.update.ID.Get(); ok {
			c.Check(user.ID.MustGet(), Equals, id, Commentf("%d", i))
		}

		if name, ok := t.update.Name.Get(); ok {
			c.Check(user.Name.MustGet(), Equals, name, Commentf("%d", i))
		}

		if email, ok := t.update.Email.Get(); ok {
			c.Check(user.Email.MustGet(), Equals, email, Commentf("%d", i))
		}

		if t.update.PasswordDigest != nil {
			if bytes.Compare(t.update.PasswordDigest, user.PasswordDigest) != 0 {
				c.Errorf("%d. PasswordDigest was %v, expected %v", i, user.PasswordDigest, t.update.PasswordDigest)
			}
		}

		if t.update.PasswordSalt != nil {
			if bytes.Compare(t.update.PasswordSalt, user.PasswordSalt) != 0 {
				c.Errorf("%d. PasswordSalt was %v, expected %v", i, user.PasswordSalt, t.update.PasswordSalt)
			}
		}
	}
}

func (s *RepositorySuite) TestFeeds(c *C) {
	userID, err := s.repo.CreateUser(s.newUser())
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
	userID, err := s.repo.CreateUser(s.newUser())
	c.Assert(err, IsNil)

	now := time.Now()

	url := "http://bar"
	err = s.repo.CreateSubscription(userID, url)
	c.Assert(err, IsNil)

	subscriptions, err := s.repo.GetSubscriptions(userID)
	c.Assert(err, IsNil)
	c.Assert(subscriptions, HasLen, 1)
	feedID := subscriptions[0].FeedID.MustGet()

	update := &parsedFeed{name: "baz", items: []parsedItem{
		{url: "http://baz/bar", title: "Baz", publicationTime: box.NewTime(now)},
	}}
	err = s.repo.UpdateFeedWithFetchSuccess(feedID, update, box.String{}, now)
	c.Assert(err, IsNil)

	buffer := &bytes.Buffer{}
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
	userID, err := s.repo.CreateUser(s.newUser())
	c.Assert(err, IsNil)

	now := time.Now()

	url := "http://bar"
	err = s.repo.CreateSubscription(userID, url)
	c.Assert(err, IsNil)

	subscriptions, err := s.repo.GetSubscriptions(userID)
	c.Assert(err, IsNil)
	c.Assert(subscriptions, HasLen, 1)
	feedID := subscriptions[0].FeedID.MustGet()

	update := &parsedFeed{name: "baz", items: []parsedItem{
		{url: "http://baz/bar", title: "Baz"},
	}}
	err = s.repo.UpdateFeedWithFetchSuccess(feedID, update, box.String{}, now)
	c.Assert(err, IsNil)

	buffer := &bytes.Buffer{}
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
	userID, err := s.repo.CreateUser(s.newUser())
	c.Assert(err, IsNil)

	url := "http://foo"
	err = s.repo.CreateSubscription(userID, url)
	c.Assert(err, IsNil)

	subscriptions, err := s.repo.GetSubscriptions(userID)
	c.Assert(err, IsNil)
	c.Assert(subscriptions, HasLen, 1)
	c.Assert(subscriptions[0].URL.MustGet(), Equals, url)
}

func (s *RepositorySuite) TestDeleteSubscription(c *C) {
	userID, err := s.repo.CreateUser(s.newUser())
	c.Assert(err, IsNil)

	err = s.repo.CreateSubscription(userID, "http://foo")
	c.Assert(err, IsNil)

	subscriptions, err := s.repo.GetSubscriptions(userID)
	c.Assert(err, IsNil)
	c.Assert(subscriptions, HasLen, 1)
	feedID := subscriptions[0].FeedID.MustGet()

	update := &parsedFeed{name: "baz", items: []parsedItem{
		{url: "http://baz/bar", title: "Baz", publicationTime: box.NewTime(time.Now())},
	}}
	err = s.repo.UpdateFeedWithFetchSuccess(feedID, update, box.String{}, time.Now().Add(-20*time.Minute))
	c.Assert(err, IsNil)

	err = s.repo.DeleteSubscription(userID, feedID)
	c.Assert(err, IsNil)

	subscriptions, err = s.repo.GetSubscriptions(userID)
	c.Assert(err, IsNil)
	c.Check(subscriptions, HasLen, 0)

	// feed should have been deleted as it was the last user
	staleFeeds, err := s.repo.GetFeedsUncheckedSince(time.Now())
	c.Assert(err, IsNil)
	c.Check(staleFeeds, HasLen, 0)
}

func (s *RepositorySuite) TestCopySubscriptionsForUserAsJSON(c *C) {
	userID, err := s.repo.CreateUser(s.newUser())
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
	userID, err := s.repo.CreateUser(s.newUser())
	c.Assert(err, IsNil)

	sessionID := []byte("deadbeef")

	err = s.repo.CreateSession(sessionID, userID)
	c.Assert(err, IsNil)

	user, err := s.repo.GetUserBySessionID(sessionID)
	c.Assert(err, IsNil)
	c.Check(user.ID.MustGet(), Equals, userID)

	err = s.repo.DeleteSession(sessionID)
	c.Assert(err, IsNil)

	_, err = s.repo.GetUserBySessionID(sessionID)
	c.Assert(err, Equals, notFound)

	err = s.repo.DeleteSession(sessionID)
	c.Assert(err, Equals, notFound)
}

func (s *RepositorySuite) TestResetPasswordsLifeCycle(c *C) {
	input := &PasswordReset{
		Token:       box.NewString("token"),
		Email:       box.NewString("test@example.com"),
		RequestIP:   box.NewString("127.0.0.1"),
		RequestTime: box.NewTime(time.Date(2014, time.May, 30, 16, 10, 0, 0, time.Local)),
	}
	err := s.repo.CreatePasswordReset(input)
	c.Assert(err, IsNil)

	reset, err := s.repo.GetPasswordReset(input.Token.MustGet())
	c.Assert(err, IsNil)
	c.Check(reset.Token.GetCoerceNil(), Equals, input.Token.GetCoerceNil())
	c.Check(reset.Email.GetCoerceNil(), Equals, input.Email.GetCoerceNil())
	c.Check(reset.RequestIP.GetCoerceNil(), Equals, input.RequestIP.GetCoerceNil())
	c.Check(reset.RequestTime.GetCoerceNil(), Equals, input.RequestTime.GetCoerceNil())
	if _, present := reset.CompletionTime.Get(); present {
		c.Error("CompletionTime should have been empty, but wasn't")
	}
	if _, present := reset.CompletionIP.Get(); present {
		c.Error("CompletionIP should have been empty, but wasn't")
	}

	update := &PasswordReset{
		CompletionIP:   box.NewString("192.168.0.2"),
		CompletionTime: box.NewTime(time.Date(2014, time.May, 30, 16, 15, 0, 0, time.Local)),
	}

	err = s.repo.UpdatePasswordReset(input.Token.MustGet(), update)
	c.Assert(err, IsNil)

	reset, err = s.repo.GetPasswordReset(input.Token.MustGet())
	c.Assert(err, IsNil)
	c.Check(reset.Token.GetCoerceNil(), Equals, input.Token.GetCoerceNil())
	c.Check(reset.Email.GetCoerceNil(), Equals, input.Email.GetCoerceNil())
	c.Check(reset.RequestIP.GetCoerceNil(), Equals, input.RequestIP.GetCoerceNil())
	c.Check(reset.RequestTime.GetCoerceNil(), Equals, input.RequestTime.GetCoerceNil())
	c.Check(reset.CompletionIP.GetCoerceNil(), Equals, update.CompletionIP.GetCoerceNil())
	c.Check(reset.CompletionTime.GetCoerceNil(), Equals, update.CompletionTime.GetCoerceNil())
}
