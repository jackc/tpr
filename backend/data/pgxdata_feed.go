package data

import (
	"github.com/jackc/pgx/v5/pgtype"
)

type Feed struct {
	ID              pgtype.Int4
	Name            pgtype.Text
	URL             pgtype.Text
	LastFetchTime   pgtype.Timestamptz
	ETag            pgtype.Text
	LastFailure     pgtype.Text
	LastFailureTime pgtype.Timestamptz
	FailureCount    pgtype.Int4
	CreationTime    pgtype.Timestamptz
}
