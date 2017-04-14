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

func selectUser(db Queryer, name, sql string, arg interface{}) (*User, error) {
	user := User{}

	err := prepareQueryRow(db, name, sql, arg).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordDigest, &user.PasswordSalt)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	return &user, nil
}

const getUserByNameSQL = `select id, name, email, password_digest, password_salt from users where name=$1`

func SelectUserByName(db Queryer, name string) (*User, error) {
	return selectUser(db, "getUserByName", getUserByNameSQL, name)
}

const getUserByEmailSQL = `select id, name, email, password_digest, password_salt from users where email=$1`

func SelectUserByEmail(db Queryer, email string) (*User, error) {
	return selectUser(db, "getUserByEmail", getUserByEmailSQL, email)
}

const getUserBySessionIDSQL = `select users.id, name, email, password_digest, password_salt
from sessions
  join users on sessions.user_id=users.id
where sessions.id=$1`

func SelectUserBySessionID(db Queryer, id []byte) (*User, error) {
	return selectUser(db, "getUserBySessionID", getUserBySessionIDSQL, id)
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

	return user.ID.Int, nil
}
