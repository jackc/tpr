package data

import (
	"fmt"
	"strings"

	"github.com/jackc/pgx"
)

type DuplicationError struct {
	Field string // Field or fields that caused the rejection
}

func (e DuplicationError) Error() string {
	return fmt.Sprintf("%s is already taken", e.Field)
}

func selectUser(db Queryer, sql string, arg interface{}) (*User, error) {
	user := User{}

	err := db.QueryRow(sql, arg).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordDigest, &user.PasswordSalt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

func SelectUserByName(db Queryer, name string) (*User, error) {
	return selectUser(db, "getUserByName", name)
}

func SelectUserByEmail(db Queryer, email string) (*User, error) {
	return selectUser(db, "getUserByEmail", email)
}

func SelectUserBySessionID(db Queryer, id []byte) (*User, error) {
	return selectUser(db, "getUserBySessionID", id)
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
