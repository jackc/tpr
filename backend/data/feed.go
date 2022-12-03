package data

import (
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

type Feed struct {
	ID              int32
	Name            string
	URL             string
	LastFetchTime   pgtype.Timestamptz
	ETag            pgtype.Text
	LastFailure     pgtype.Text
	LastFailureTime pgtype.Timestamptz
	FailureCount    int32
	CreationTime    time.Time
}
