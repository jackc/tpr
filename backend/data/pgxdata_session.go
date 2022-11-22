package data

import (
	"context"
	"strings"

	"github.com/jackc/pgsql"
	"github.com/jackc/pgx/v5/pgtype"
)

type Session struct {
	ID        []byte
	UserID    pgtype.Int4
	StartTime pgtype.Timestamptz
}

func InsertSession(ctx context.Context, db Queryer, row *Session) error {
	args := pgsql.Args{}

	var columns, values []string

	columns = append(columns, `id`)
	values = append(values, args.Use(&row.ID).String())
	columns = append(columns, `user_id`)
	values = append(values, args.Use(&row.UserID).String())

	sql := `insert into "sessions"(` + strings.Join(columns, ", ") + `)
values(` + strings.Join(values, ",") + `)
returning "id"
  `

	return db.QueryRow(ctx, sql, args.Values()...).Scan(&row.ID)
}

func DeleteSession(ctx context.Context, db Queryer,
	id []byte,
) error {
	args := pgsql.Args{}

	sql := `delete from "sessions" where ` + `"id"=` + args.Use(id).String()

	commandTag, err := db.Exec(ctx, sql, args.Values()...)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}
