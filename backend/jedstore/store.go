// Package jedstore adapts one shared JED database to TPR's SQL data layer.
package jedstore

import (
	"context"
	"database/sql/driver"
	_ "embed"
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"sync"

	jed "github.com/jackc/jed/impl/go"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/tpr/backend/data"
)

//go:embed schema.sql
var schema string

type Store struct {
	db *jed.Database
	mu sync.Mutex
}

func Open(path string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, err
	}
	var db *jed.Database
	var err error
	if _, err = os.Stat(path); errors.Is(err, os.ErrNotExist) {
		db, err = jed.CreateDatabase(jed.CreateOptions{Path: path})
	} else if err == nil {
		db, err = jed.OpenDatabase(path)
	}
	if err != nil {
		return nil, err
	}
	s := &Store{db: db}
	err = s.WithinTx(context.Background(), func(q data.DB) error {
		if _, err := q.Exec(context.Background(), `create table if not exists schema_version(version integer not null)`); err != nil {
			return err
		}
		var version int
		err := q.QueryRow(context.Background(), `select version from schema_version`).Scan(&version)
		if err == nil {
			if version != 1 {
				return fmt.Errorf("unsupported JED schema version %d", version)
			}
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		for stmt := range jed.SplitStatements(schema) {
			if _, err := q.Exec(context.Background(), stmt.Text); err != nil {
				return fmt.Errorf("JED schema %s: %w", stmt.Text, err)
			}
		}
		_, err = q.Exec(context.Background(), `insert into schema_version values (1)`)
		return err
	})
	if err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}
func (s *Store) Close() error { s.mu.Lock(); defer s.mu.Unlock(); return s.db.Close() }
func (s *Store) WithinTx(ctx context.Context, fn func(data.DB) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	session := s.db.Session(jed.SessionOptions{})
	defer session.Close()
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := session.Begin(true); err != nil {
		return translate(err)
	}
	defer session.Rollback()
	if err := fn(queryer{session}); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return translate(session.Commit())
}
func (s *Store) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return queryer{s.db}.Exec(ctx, sql, args...)
}
func (s *Store) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return queryer{s.db}.Query(ctx, sql, args...)
}
func (s *Store) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	r, e := s.Query(ctx, sql, args...)
	return row{r, e}
}

type queryer struct{ q jed.Queryer }

func translate(err error) error {
	var e *jed.EngineError
	if errors.As(err, &e) {
		return &pgconn.PgError{Code: e.Code(), Message: e.Error(), ConstraintName: e.ConstraintName}
	}
	return err
}
func argsFor(args []any) ([]any, error) {
	out := make([]any, len(args))
	for i, a := range args {
		if a == nil {
			continue
		}
		v := reflect.ValueOf(a)
		for v.Kind() == reflect.Pointer {
			if v.IsNil() {
				a = nil
				break
			}
			v = v.Elem()
			a = v.Interface()
		}
		switch x := a.(type) {
		case netip.Addr:
			if x.IsValid() {
				out[i] = x.String()
			}
		case driver.Valuer:
			var err error
			out[i], err = x.Value()
			if err != nil {
				return nil, err
			}
		default:
			out[i] = a
		}
	}
	return out, nil
}
func (q queryer) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	a, e := argsFor(args)
	if e != nil {
		return pgconn.CommandTag{}, e
	}
	r, e := q.q.Exec(ctx, sql, a...)
	n, _ := r.RowsAffected()
	verb := ""
	if words := strings.Fields(sql); len(words) > 0 {
		verb = strings.ToUpper(words[0])
	}
	return pgconn.NewCommandTag(fmt.Sprintf("%s %d", verb, n)), translate(e)
}
func (q queryer) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	failed := func(err error) (pgx.Rows, error) {
		return data.NewBufferedRows(nil, nil, pgconn.CommandTag{}, err), err
	}
	a, err := argsFor(args)
	if err != nil {
		return failed(err)
	}
	r, err := q.q.Query(ctx, sql, a...)
	if err != nil {
		return failed(translate(err))
	}
	defer r.Close()
	fields := make([]pgconn.FieldDescription, len(r.ColumnNames()))
	m := pgtype.NewMap()
	aliases := map[string]string{"decimal": "numeric", "i16": "int2", "i32": "int4", "i64": "int8", "f32": "float4", "f64": "float8", "integer": "int4", "bigint": "int8", "boolean": "bool", "timestamp with time zone": "timestamptz", "character varying": "varchar"}
	for i, name := range r.ColumnNames() {
		typ := r.ColumnTypes()[i]
		if alias := aliases[typ]; alias != "" {
			typ = alias
		}
		t, ok := m.TypeForName(typ)
		if !ok {
			return failed(fmt.Errorf("unsupported JED result type %q", typ))
		}
		fields[i] = pgconn.FieldDescription{Name: name, DataTypeOID: t.OID}
	}
	var values [][][]byte
	for r.Next() {
		row := make([][]byte, len(fields))
		for i := range row {
			v := r.Value(i)
			if v.Kind != jed.ValNull {
				row[i] = []byte(v.Render())
			}
		}
		values = append(values, row)
	}
	err = translate(r.Err())
	n, _ := r.RowsAffected()
	if n < int64(len(values)) {
		n = int64(len(values))
	}
	return data.NewBufferedRows(fields, values, pgconn.NewCommandTag(fmt.Sprintf("SELECT %d", n)), err), err
}
func (q queryer) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	r, e := q.Query(ctx, sql, args...)
	return row{r, e}
}

type row struct {
	rows pgx.Rows
	err  error
}

func (r row) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	defer r.rows.Close()
	if !r.rows.Next() {
		if e := r.rows.Err(); e != nil {
			return e
		}
		return pgx.ErrNoRows
	}
	return r.rows.Scan(dest...)
}
