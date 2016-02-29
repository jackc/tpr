package data

import (
	"fmt"
	"strings"
)

type DuplicationError struct {
	Field string // Field or fields that caused the rejection
}

func (e DuplicationError) Error() string {
	return fmt.Sprintf("%s is already taken", e.Field)
}

func CreateUser(db Queryer, user *User) (int32, error) {
	err := InsertUser(db, user)
	if err != nil {
		if strings.Contains(err.Error(), "users_name_unq") {
			return 0, DuplicationError{Field: "name"}
		}
		if strings.Contains(err.Error(), "users_email_key") {
			return 0, DuplicationError{Field: "email"}
		}
		return 0, err
	}

	return user.ID.Value, nil
}
