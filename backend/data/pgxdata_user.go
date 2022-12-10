package data

import (
	"context"
	"strings"

	"errors"

	"github.com/jackc/pgsql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
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
	sets := make([]string, 0, 5)
	args := pgsql.Args{}

	sets = append(sets, `name`+"="+args.Use(&row.Name).String())
	sets = append(sets, `password_digest`+"="+args.Use(&row.PasswordDigest).String())
	sets = append(sets, `password_salt`+"="+args.Use(&row.PasswordSalt).String())
	sets = append(sets, `email`+"="+args.Use(&row.Email).String())

	if len(sets) == 0 {
		return nil
	}

	sql := `update "users" set ` + strings.Join(sets, ", ") + ` where ` + `"id"=` + args.Use(id).String()

	commandTag, err := db.Exec(ctx, sql, args.Values()...)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}
