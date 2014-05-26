package main

import (
	"errors"
	"github.com/JackC/box"
	"io"
	"sync"
	"time"
)

type int32Seq struct {
	current int32
	mutex   sync.Mutex
}

func (s *int32Seq) next() int32 {
	s.mutex.Lock()
	s.current++
	n := s.current
	s.mutex.Unlock()
	return n
}

type memorySubscription struct {
	userID int32
	feedID int32
}

type memoryRepository struct {
	mutex         sync.Mutex
	usersByID     map[int32]*User
	usersIDSeq    int32Seq
	usersByName   map[string]*User
	usersByEmail  map[string]*User
	sessions      map[string]int32
	feedsIDSeq    int32Seq
	feedsByID     map[int32]*Feed
	feedsByURL    map[string]*Feed
	subscriptions []memorySubscription
}

func NewMemoryRepository() (*memoryRepository, error) {
	repo := &memoryRepository{}
	repo.usersByID = make(map[int32]*User)
	repo.usersByName = make(map[string]*User)
	repo.usersByEmail = make(map[string]*User)
	repo.sessions = make(map[string]int32)
	repo.feedsByID = make(map[int32]*Feed)
	repo.feedsByURL = make(map[string]*Feed)
	return repo, nil
}

func copyUser(src *User) *User {
	user := &User{}
	user.ID = src.ID
	user.Name = src.Name
	user.Email = src.Email
	user.PasswordDigest = make([]byte, len(src.PasswordDigest))
	copy(user.PasswordDigest, src.PasswordDigest)
	user.PasswordSalt = make([]byte, len(src.PasswordSalt))
	copy(user.PasswordSalt, src.PasswordSalt)
	return user
}

func (repo *memoryRepository) indexUser(user *User) error {
	if _, ok := repo.usersByName[user.Name.MustGet()]; ok {
		return DuplicationError{Field: "name"}
	}
	if email, ok := user.Email.Get(); ok {
		if _, ok := repo.usersByEmail[email]; ok {
			return DuplicationError{Field: "email"}
		}
	}

	repo.usersByID[user.ID.MustGet()] = user
	repo.usersByName[user.Name.MustGet()] = user

	if email, ok := user.Email.Get(); ok {
		repo.usersByEmail[email] = user
	}

	return nil
}

func (repo *memoryRepository) deindexUser(user *User) {
	delete(repo.usersByID, user.ID.MustGet())
	delete(repo.usersByName, user.Name.MustGet())
	if email, ok := user.Email.Get(); ok {
		delete(repo.usersByEmail, email)
	}
}

func (repo *memoryRepository) CreateUser(src *User) (int32, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	user := copyUser(src)
	user.ID.Set(repo.usersIDSeq.next())

	err := repo.indexUser(user)
	if err != nil {
		return 0, err
	}

	return user.ID.MustGet(), nil
}

func (repo *memoryRepository) GetUser(userID int32) (*User, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	src, ok := repo.usersByID[userID]
	if !ok {
		return nil, notFound
	}

	user := copyUser(src)

	return user, nil
}

func (repo *memoryRepository) GetUserByName(name string) (*User, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	src, ok := repo.usersByName[name]
	if !ok {
		return nil, notFound
	}

	user := copyUser(src)

	return user, nil
}

func (repo *memoryRepository) UpdateUser(userID int32, attributes *User) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	origUser, ok := repo.usersByID[userID]
	if !ok {
		return notFound
	}

	user := copyUser(origUser)

	if _, ok := attributes.ID.Get(); ok {
		user.ID = attributes.ID
	}
	if _, ok := attributes.Name.Get(); ok {
		user.Name = attributes.Name
	}
	if _, ok := attributes.Email.Get(); ok {
		user.Email = attributes.Email
	}
	if attributes.PasswordDigest != nil {
		user.PasswordDigest = make([]byte, len(attributes.PasswordDigest))
		copy(user.PasswordDigest, attributes.PasswordDigest)
	}
	if attributes.PasswordSalt != nil {
		user.PasswordSalt = make([]byte, len(attributes.PasswordSalt))
		copy(user.PasswordSalt, attributes.PasswordSalt)
	}

	repo.deindexUser(origUser)

	err := repo.indexUser(user)
	if err != nil {
		repo.indexUser(origUser) // this shouldn't be able to fail because it was already indexed
		return err
	}

	return nil
}

func (repo *memoryRepository) GetFeedsUncheckedSince(since time.Time) (feeds []Feed, err error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return nil, errors.New("Not implemented")
}

func (repo *memoryRepository) UpdateFeedWithFetchSuccess(feedID int32, update *parsedFeed, etag box.String, fetchTime time.Time) (err error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return errors.New("Not implemented")
}

func (repo *memoryRepository) UpdateFeedWithFetchUnchanged(feedID int32, fetchTime time.Time) (err error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return errors.New("Not implemented")
}

func (repo *memoryRepository) UpdateFeedWithFetchFailure(feedID int32, failure string, fetchTime time.Time) (err error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return errors.New("Not implemented")
}

func (repo *memoryRepository) CopySubscriptionsForUserAsJSON(w io.Writer, userID int32) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return errors.New("Not implemented")
}

func (repo *memoryRepository) CopyUnreadItemsAsJSONByUserID(w io.Writer, userID int32) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return errors.New("Not implemented")
}

func (repo *memoryRepository) MarkItemRead(userID, itemID int32) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	return errors.New("Not implemented")
}

func (repo *memoryRepository) CreateSubscription(userID int32, feedURL string) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	_, ok := repo.usersByID[userID]
	if !ok {
		return notFound
	}

	feed, ok := repo.feedsByURL[feedURL]
	if !ok {
		feed = &Feed{}
		feed.ID.Set(repo.feedsIDSeq.next())
		feed.Name.Set(feedURL)
		feed.URL.Set(feedURL)
		feed.CreationTime.Set(time.Now())

		repo.feedsByID[feed.ID.MustGet()] = feed
		repo.feedsByURL[feed.URL.MustGet()] = feed
	}

	// TODO - One user can't subscribe to same feed twice

	repo.subscriptions = append(repo.subscriptions, memorySubscription{userID: userID, feedID: feed.ID.MustGet()})

	return nil
}

func (repo *memoryRepository) GetSubscriptions(userID int32) ([]Subscription, error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	subs := make([]Subscription, 0, 16)

	for _, rs := range repo.subscriptions {
		if rs.userID == userID {
			feed := repo.feedsByID[rs.feedID]
			s := Subscription{}
			s.FeedID = feed.ID
			s.Name = feed.Name
			s.URL = feed.URL
			s.LastFetchTime = feed.LastFetchTime
			s.LastFailure = feed.LastFailure
			s.LastFailureTime = feed.LastFailureTime
			s.FailureCount = feed.FailureCount

			// TODO - ItemCount
			// TODO - LastPublicationTime

			subs = append(subs, s)
		}
	}

	return subs, nil
}

func (repo *memoryRepository) DeleteSubscription(userID, feedID int32) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	for i, s := range repo.subscriptions {
		if s.userID == userID && s.feedID == feedID {
			repo.subscriptions[i] = repo.subscriptions[len(repo.subscriptions)-1]
			repo.subscriptions = repo.subscriptions[:len(repo.subscriptions)-1]
			return nil
		}
	}

	return notFound
}

func (repo *memoryRepository) CreateSession(id []byte, userID int32) (err error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	repo.sessions[string(id)] = userID

	return nil
}

func (repo *memoryRepository) GetUserIDBySessionID(id []byte) (userID int32, err error) {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	userID, ok := repo.sessions[string(id)]
	if !ok {
		return 0, notFound
	}

	return userID, nil
}

func (repo *memoryRepository) DeleteSession(id []byte) error {
	repo.mutex.Lock()
	defer repo.mutex.Unlock()

	_, ok := repo.sessions[string(id)]
	if !ok {
		return notFound
	}

	delete(repo.sessions, string(id))

	return nil
}
