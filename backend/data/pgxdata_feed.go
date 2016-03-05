package data

// This file is automatically generated by pgxdata.

import (
	"strings"

	"github.com/jackc/pgx"
)

type Feed struct {
	ID              Int32
	Name            String
	URL             String
	LastFetchTime   Time
	ETag            String
	LastFailure     String
	LastFailureTime Time
	FailureCount    Int32
	CreationTime    Time
}

const countFeedSQL = `select count(*) from "feeds"`

func CountFeed(db Queryer) (int64, error) {
	var n int64
	err := prepareQueryRow(db, "pgxdataCountFeed", countFeedSQL).Scan(&n)
	return n, err
}

const SelectAllFeedSQL = `select
  "id",
  "name",
  "url",
  "last_fetch_time",
  "etag",
  "last_failure",
  "last_failure_time",
  "failure_count",
  "creation_time"
from "feeds"`

func SelectAllFeed(db Queryer) ([]Feed, error) {
	var rows []Feed

	dbRows, err := prepareQuery(db, "pgxdataSelectAllFeed", SelectAllFeedSQL)
	if err != nil {
		return nil, err
	}

	for dbRows.Next() {
		var row Feed
		dbRows.Scan(
			&row.ID,
			&row.Name,
			&row.URL,
			&row.LastFetchTime,
			&row.ETag,
			&row.LastFailure,
			&row.LastFailureTime,
			&row.FailureCount,
			&row.CreationTime,
		)
		rows = append(rows, row)
	}

	if dbRows.Err() != nil {
		return nil, dbRows.Err()
	}

	return rows, nil
}

const selectFeedByPKSQL = `select
  "id",
  "name",
  "url",
  "last_fetch_time",
  "etag",
  "last_failure",
  "last_failure_time",
  "failure_count",
  "creation_time"
from "feeds"
where "id"=$1`

func SelectFeedByPK(
	db Queryer,
	id int32,
) (*Feed, error) {
	var row Feed
	err := prepareQueryRow(db, "pgxdataSelectFeedByPK", selectFeedByPKSQL, id).Scan(
		&row.ID,
		&row.Name,
		&row.URL,
		&row.LastFetchTime,
		&row.ETag,
		&row.LastFailure,
		&row.LastFailureTime,
		&row.FailureCount,
		&row.CreationTime,
	)
	if err == pgx.ErrNoRows {
		return nil, ErrNotFound
	} else if err != nil {
		return nil, err
	}

	return &row, nil
}

func InsertFeed(db Queryer, row *Feed) error {
	args := pgx.QueryArgs(make([]interface{}, 0, 9))

	var columns, values []string

	row.ID.addInsert(`id`, &columns, &values, &args)
	row.Name.addInsert(`name`, &columns, &values, &args)
	row.URL.addInsert(`url`, &columns, &values, &args)
	row.LastFetchTime.addInsert(`last_fetch_time`, &columns, &values, &args)
	row.ETag.addInsert(`etag`, &columns, &values, &args)
	row.LastFailure.addInsert(`last_failure`, &columns, &values, &args)
	row.LastFailureTime.addInsert(`last_failure_time`, &columns, &values, &args)
	row.FailureCount.addInsert(`failure_count`, &columns, &values, &args)
	row.CreationTime.addInsert(`creation_time`, &columns, &values, &args)

	sql := `insert into "feeds"(` + strings.Join(columns, ", ") + `)
values(` + strings.Join(values, ",") + `)
returning "id"
  `

	psName := preparedName("pgxdataInsertFeed", sql)

	return prepareQueryRow(db, psName, sql, args...).Scan(&row.ID)
}

func UpdateFeed(db Queryer,
	id int32,
	row *Feed,
) error {
	sets := make([]string, 0, 9)
	args := pgx.QueryArgs(make([]interface{}, 0, 9))

	row.ID.addUpdate(`id`, &sets, &args)
	row.Name.addUpdate(`name`, &sets, &args)
	row.URL.addUpdate(`url`, &sets, &args)
	row.LastFetchTime.addUpdate(`last_fetch_time`, &sets, &args)
	row.ETag.addUpdate(`etag`, &sets, &args)
	row.LastFailure.addUpdate(`last_failure`, &sets, &args)
	row.LastFailureTime.addUpdate(`last_failure_time`, &sets, &args)
	row.FailureCount.addUpdate(`failure_count`, &sets, &args)
	row.CreationTime.addUpdate(`creation_time`, &sets, &args)

	if len(sets) == 0 {
		return nil
	}

	sql := `update "feeds" set ` + strings.Join(sets, ", ") + ` where ` + `"id"=` + args.Append(id)

	psName := preparedName("pgxdataUpdateFeed", sql)

	commandTag, err := prepareExec(db, psName, sql, args...)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}

func DeleteFeed(db Queryer,
	id int32,
) error {
	args := pgx.QueryArgs(make([]interface{}, 0, 1))

	sql := `delete from "feeds" where ` + `"id"=` + args.Append(id)

	commandTag, err := prepareExec(db, "pgxdataDeleteFeed", sql, args...)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() != 1 {
		return ErrNotFound
	}
	return nil
}
