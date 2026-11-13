package data

import (
	"context"
	"time"

	"github.com/jackc/pgxutil"
)

type Session struct {
	ID     []byte
	UserID string
}

func InsertSession(ctx context.Context, db DB, row *Session) error {
	_, err := db.Exec(ctx, `insert into sessions (id, user_id, start_time) values ($1, $2, $3)`, row.ID, row.UserID, time.Now().UTC().Truncate(time.Microsecond))
	return err
}

func DeleteSession(ctx context.Context, db DB,
	id []byte,
) error {
	_, err := pgxutil.ExecRow(ctx, db, `delete from sessions where id = $1`, id)
	if err != nil {
		return err
	}

	return nil
}

// DeleteSessionByUserID only revokes sessions owned by the authenticated reader.
// A missing or foreign session is a no-op.
func DeleteSessionByUserID(ctx context.Context, db DB, id []byte, userID string) error {
	_, err := db.Exec(ctx, `delete from sessions where id=$1 and user_id=$2`, id, userID)
	return err
}
