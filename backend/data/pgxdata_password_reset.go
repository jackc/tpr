package data

import (
	"context"
	"net/netip"
	"strings"

	"errors"

	"github.com/jackc/pgsql"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type PasswordReset struct {
	Token          pgtype.Text
	Email          pgtype.Text
	RequestIP      netip.Addr
	RequestTime    pgtype.Timestamptz
	UserID         pgtype.Int4
	CompletionIP   netip.Addr
	CompletionTime pgtype.Timestamptz
}

const selectPasswordResetByPKSQL = `select
  "token",
  "email",
  "request_ip",
  "request_time",
  "user_id",
  "completion_ip",
  "completion_time"
from "password_resets"
where "token"=$1`

func SelectPasswordResetByPK(
	ctx context.Context,
	db Queryer,
	token string,
) (*PasswordReset, error) {
	var row PasswordReset
	err := db.QueryRow(ctx, selectPasswordResetByPKSQL, token).Scan(
		&row.Token,
		&row.Email,
		&row.RequestIP,
		&row.RequestTime,
		&row.UserID,
		&row.CompletionIP,
		&row.CompletionTime,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return &row, nil
}

func InsertPasswordReset(ctx context.Context, db Queryer, row *PasswordReset) error {
	args := pgsql.Args{}

	var columns, values []string

	columns = append(columns, `token`)
	values = append(values, args.Use(&row.Token).String())
	columns = append(columns, `email`)
	values = append(values, args.Use(&row.Email).String())
	columns = append(columns, `request_ip`)
	values = append(values, args.Use(&row.RequestIP).String())
	columns = append(columns, `request_time`)
	values = append(values, args.Use(&row.RequestTime).String())
	columns = append(columns, `user_id`)
	values = append(values, args.Use(&row.UserID).String())
	columns = append(columns, `completion_ip`)
	values = append(values, args.Use(&row.CompletionIP).String())
	columns = append(columns, `completion_time`)
	values = append(values, args.Use(&row.CompletionTime).String())

	sql := `insert into "password_resets"(` + strings.Join(columns, ", ") + `)
values(` + strings.Join(values, ",") + `)
returning "token"
  `

	return db.QueryRow(ctx, sql, args.Values()...).Scan(&row.Token)
}
