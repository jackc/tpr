package data

import (
	"context"
	"net/netip"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgxutil"
)

type PasswordReset struct {
	Token          string
	Email          string
	RequestIP      netip.Addr
	RequestTime    time.Time
	UserID         pgtype.Int4
	CompletionIP   netip.Addr
	CompletionTime pgtype.Timestamptz
}

const selectPasswordResetSQL = `select token, email, request_ip, request_time, user_id, completion_ip, completion_time from password_resets`
const selectPasswordResetByPKSQL = selectPasswordResetSQL + ` where token = $1`

func RowToAddrOfPasswordReset(row pgx.CollectableRow) (*PasswordReset, error) {
	pr := &PasswordReset{}
	err := row.Scan(&pr.Token, &pr.Email, &pr.RequestIP, &pr.RequestTime, &pr.UserID, &pr.CompletionIP, &pr.CompletionTime)
	return pr, err
}

func SelectPasswordResetByPK(
	ctx context.Context,
	db pgxutil.DB,
	token string,
) (*PasswordReset, error) {
	rows, _ := db.Query(ctx, selectPasswordResetByPKSQL, token)
	pr, err := pgx.CollectOneRow(rows, RowToAddrOfPasswordReset)
	if err != nil {
		return nil, err
	}

	return pr, nil
}
