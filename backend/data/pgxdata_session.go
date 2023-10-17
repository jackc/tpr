package data

import (
	"context"

	"github.com/jackc/pgxutil"
)

type Session struct {
	ID     []byte
	UserID int32
}

func InsertSession(ctx context.Context, db pgxutil.DB, row *Session) error {
	_, err := db.Exec(ctx, `insert into sessions (id, user_id) values ($1, $2)`, row.ID, row.UserID)
	return err
}

func DeleteSession(ctx context.Context, db pgxutil.DB,
	id []byte,
) error {
	_, err := pgxutil.ExecRow(ctx, db, `delete from sessions where id = $1`, id)
	if err != nil {
		return err
	}

	return nil
}
