// Package dualstore is a fail-stop validation harness: PostgreSQL supplies
// results, while every statement also runs on JED and is checked for divergence.
package dualstore

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"reflect"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/tpr/backend/data"
)

type state struct {
	mu     sync.Mutex
	failed bool
}
type Store struct {
	primary, secondary data.DB
	state              *state
	inTx               bool
}

func New(primary, secondary data.DB) *Store {
	return &Store{primary: primary, secondary: secondary, state: &state{}}
}

// A recovered HTTP panic must not allow further operations after divergence.
func (s *Store) guard() func() {
	if !s.inTx {
		s.state.mu.Lock()
	}
	if s.state.failed {
		if !s.inTx {
			s.state.mu.Unlock()
		}
		panic("dualstore: disabled after divergence; reconcile stores before restarting")
	}
	return func() {
		if !s.inTx {
			defer s.state.mu.Unlock()
		}
		if p := recover(); p != nil {
			s.state.failed = true
			panic(p)
		}
	}
}
func (s *Store) WithinTx(ctx context.Context, fn func(data.DB) error) error {
	defer s.guard()()
	readyToCommit := false
	err := data.WithinTx(ctx, s.primary, func(p data.DB) error {
		return data.WithinTx(ctx, s.secondary, func(q data.DB) error {
			err := fn(&Store{primary: p, secondary: q, state: s.state, inTx: true})
			readyToCommit = err == nil
			return err
		})
	})
	if err != nil && readyToCommit {
		panic(fmt.Sprintf("dualstore: commit failed; reconcile stores: %v", err))
	}
	return err
}
func checkErrors(a, b error) {
	if a == nil && b == nil {
		return
	}
	var pa, pb *pgconn.PgError
	if errors.As(a, &pa) && errors.As(b, &pb) && pa.Code == pb.Code && pa.ConstraintName == pb.ConstraintName {
		return
	}
	if errors.Is(a, context.Canceled) && errors.Is(b, context.Canceled) {
		return
	}
	if errors.Is(a, context.DeadlineExceeded) && errors.Is(b, context.DeadlineExceeded) {
		return
	}
	panic(fmt.Sprintf("dualstore: error divergence (PostgreSQL: %v; JED: %v)", a, b))
}
func (s *Store) Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error) {
	defer s.guard()()
	a, ae := s.primary.Exec(ctx, sql, args...)
	b, be := s.secondary.Exec(ctx, sql, args...)
	checkErrors(ae, be)
	if ae == nil && a.RowsAffected() != b.RowsAffected() {
		panic("dualstore: affected row count divergence")
	}
	return a, ae
}
func (s *Store) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	defer s.guard()()
	ar, ae := s.primary.Query(ctx, sql, args...)
	a, av, ae := data.BufferRows(ar, ae)
	br, be := s.secondary.Query(ctx, sql, args...)
	b, bv, be := data.BufferRows(br, be)
	checkErrors(ae, be)
	if ae == nil {
		af, bf := a.FieldDescriptions(), b.FieldDescriptions()
		if len(af) != len(bf) {
			panic("dualstore: result column count divergence")
		}
		for i := range af {
			if af[i].Name != bf[i].Name {
				panic("dualstore: result column name divergence")
			}
		}
		if a.CommandTag().RowsAffected() != b.CommandTag().RowsAffected() {
			panic("dualstore: query affected row count divergence")
		}
		aj, err := json.Marshal(data.CanonicalValue(av))
		if err != nil {
			panic(err)
		}
		bj, err := json.Marshal(data.CanonicalValue(bv))
		if err != nil {
			panic(err)
		}
		var x, y any
		ad := json.NewDecoder(bytes.NewReader(aj))
		ad.UseNumber()
		if err := ad.Decode(&x); err != nil {
			panic(err)
		}
		bd := json.NewDecoder(bytes.NewReader(bj))
		bd.UseNumber()
		if err := bd.Decode(&y); err != nil {
			panic(err)
		}
		if !equalJSON(x, y) {
			panic("dualstore: result divergence")
		}

	}
	return a, ae
}
func (s *Store) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	r, e := s.Query(ctx, sql, args...)
	return row{r, e}
}

type row struct {
	r pgx.Rows
	e error
}

func (r row) Scan(dest ...any) error {
	if r.e != nil {
		return r.e
	}
	defer r.r.Close()
	if !r.r.Next() {
		if e := r.r.Err(); e != nil {
			return e
		}
		return pgx.ErrNoRows
	}
	return r.r.Scan(dest...)
}

// Verify checks every application row before dual mode begins serving traffic.
func (s *Store) Verify(ctx context.Context) error {
	for _, t := range data.Tables {
		r, e := s.Query(ctx, "select "+t.Columns+" from "+t.Name+" order by "+t.Order)
		if e != nil {
			return e
		}
		r.Close()
	}
	return nil
}

func equalJSON(a, b any) bool {
	switch x := a.(type) {
	case json.Number:
		y, ok := b.(json.Number)
		if !ok {
			return false
		}
		rx, ok := new(big.Rat).SetString(string(x))
		if !ok {
			return false
		}
		ry, ok := new(big.Rat).SetString(string(y))
		return ok && rx.Cmp(ry) == 0
	case []any:
		y, ok := b.([]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for i, v := range x {
			if !equalJSON(v, y[i]) {
				return false
			}
		}
		return true
	case map[string]any:
		y, ok := b.(map[string]any)
		if !ok || len(x) != len(y) {
			return false
		}
		for k, v := range x {
			w, ok := y[k]
			if !ok || !equalJSON(v, w) {
				return false
			}
		}
		return true
	default:
		return reflect.DeepEqual(a, b)
	}
}
