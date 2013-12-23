package main

import (
	"bytes"
	"testing"
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
