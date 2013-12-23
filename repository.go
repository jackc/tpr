package main

type repository interface {
	createUser(name string, passwordDigest, passwordSalt []byte) (userID int32, err error)
	getUserAuthenticationByName(name string) (userID int32, passwordDigest, passwordSalt []byte, err error)
}
