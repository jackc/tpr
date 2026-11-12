package backend

import (
	"errors"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/tpr/backend/core"
	"github.com/jackc/tpr/backend/data"
)

var notFound = errors.New("not found")

func SetPassword(u *data.User, password string) error { return core.SetPassword(u, password) }
func IsPassword(u *data.User, password string) bool   { return core.IsPassword(u, password) }

type staleFeed struct {
	id   int32
	url  string
	etag string
}

func newStringFallback(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}
