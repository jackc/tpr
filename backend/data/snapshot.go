package data

import (
	"context"
	"fmt"
	"net/netip"
	"strings"
	"time"
	"uuid"

	"github.com/jackc/pgx/v5"
)

// Tables lists application columns in dependency order. Legacy IDs and migration
// metadata belong to each backend and are deliberately excluded.
var Tables = []struct {
	Name    string
	Columns string
	Order   string
}{
	{"users", "id,name,password_digest,password_salt,email", "id"},
	{"feeds", "id,name,url,last_fetch_time,etag,last_failure,last_failure_time,failure_count,creation_time", "id"},
	{"items", "id,feed_id,publication_time,title,url,creation_time", "id"},
	{"subscriptions", "user_id,feed_id", "user_id,feed_id"},
	{"unread_items", "user_id,feed_id,item_id", "user_id,feed_id,item_id"},
	{"sessions", "id,user_id,start_time", "id"},
	{"password_resets", "token,email,request_ip,request_time,user_id,completion_ip,completion_time", "token"},
}

// CanonicalValue normalizes equivalent driver representations without ignoring fields.
func CanonicalValue(v any) any {
	switch x := v.(type) {
	case [16]byte:
		return uuid.UUID(x).String()
	case time.Time:
		return x.UTC()
	case netip.Prefix:
		if x.Bits() == x.Addr().BitLen() {
			return x.Addr().String()
		}
		return x.String()
	case []any:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = CanonicalValue(v)
		}
		return out
	case [][]any:
		out := make([]any, len(x))
		for i, v := range x {
			out[i] = CanonicalValue(v)
		}
		return out
	case map[string]any:
		out := make(map[string]any, len(x))
		for k, v := range x {
			out[k] = CanonicalValue(v)
		}
		return out
	default:
		return v
	}
}

// CopySnapshot copies a stable source snapshot to an empty destination in one
// destination transaction. The caller must keep source writes stopped until cutover.
func CopySnapshot(ctx context.Context, source DB, destination DB) error {
	return WithinTx(ctx, destination, func(dest DB) error {
		for _, table := range Tables {
			var count int64
			if err := dest.QueryRow(ctx, "select count(*) from "+table.Name).Scan(&count); err != nil {
				return err
			}
			if count != 0 {
				return fmt.Errorf("destination table %s is not empty", table.Name)
			}
		}
		for _, table := range Tables {
			rows, err := source.Query(ctx, "select "+table.Columns+" from "+table.Name+" order by "+table.Order)
			if err != nil {
				return err
			}
			values, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) ([]any, error) { return r.Values() })
			if err != nil {
				return err
			}
			cols := strings.Split(table.Columns, ",")
			params := make([]string, len(cols))
			for i := range params {
				params[i] = fmt.Sprintf("$%d", i+1)
			}
			sql := "insert into " + table.Name + "(" + table.Columns + ") values (" + strings.Join(params, ",") + ")"
			for _, row := range values {
				for i, v := range row {
					row[i] = CanonicalValue(v)
				}
				if _, err := dest.Exec(ctx, sql, row...); err != nil {
					return fmt.Errorf("copy %s: %w", table.Name, err)
				}
			}
		}
		return nil
	})
}
