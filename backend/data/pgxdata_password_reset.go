package data

import (
	"context"
	"net/netip"

	"errors"

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
