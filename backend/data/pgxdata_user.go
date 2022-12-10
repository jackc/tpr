package data

import (
	"context"

	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgxrecord"
)

type User struct {
	ID             pgtype.Int4
	Name           pgtype.Text
	PasswordDigest []byte
	PasswordSalt   []byte
	Email          pgtype.Text
}

const selectUserByPKSQL = `select
  "id",
  "name",
  "password_digest",
  "password_salt",
  "email"
from "users"
where "id"=$1`

func SelectUserByPK(
	ctx context.Context,
	db Queryer,
	id int32,
) (*User, error) {
	var row User
	err := db.QueryRow(ctx, selectUserByPKSQL, id).Scan(
		&row.ID,
		&row.Name,
		&row.PasswordDigest,
		&row.PasswordSalt,
		&row.Email,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return &row, nil
}

func UpdateUser(ctx context.Context, db Queryer,
	id int32,
	row *User,
) error {
	return pgxrecord.UpdateRow(ctx, db, pgx.Identifier{"users"}, map[string]any{
		"name":            row.Name,
		"password_digest": row.PasswordDigest,
		"password_salt":   row.PasswordSalt,
		"email":           row.Email,
	}, map[string]any{
		"id": id,
	})
}
