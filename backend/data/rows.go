package data

import (
	"fmt"
	"net/netip"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// BufferedRows preserves results after the database cursor has been released.
type BufferedRows struct {
	fields []pgconn.FieldDescription
	values [][][]byte
	index  int
	err    error
	tag    pgconn.CommandTag
	closed bool
}

func (r *BufferedRows) Close()                                       { r.closed = true }
func (r *BufferedRows) Err() error                                   { return r.err }
func (r *BufferedRows) CommandTag() pgconn.CommandTag                { return r.tag }
func (r *BufferedRows) FieldDescriptions() []pgconn.FieldDescription { return r.fields }
func (r *BufferedRows) Next() bool {
	if r.closed || r.err != nil {
		return false
	}
	r.index++
	return r.index < len(r.values)
}
func (r *BufferedRows) Scan(dest ...any) error {
	if len(dest) == 1 {
		if s, ok := dest[0].(pgx.RowScanner); ok {
			return s.ScanRow(r)
		}
	}
	if len(dest) != len(r.fields) {
		return fmt.Errorf("scan destination count mismatch")
	}
	m := pgtype.NewMap()
	for i, d := range dest {
		if d == nil {
			continue
		}
		if a, ok := d.(*netip.Addr); ok && r.fields[i].DataTypeOID != pgtype.InetOID {
			v := r.values[r.index][i]
			if v == nil {
				*a = netip.Addr{}
				continue
			}
			x, e := netip.ParseAddr(string(v))
			if e != nil {
				return e
			}
			*a = x
			continue
		}
		if e := m.Scan(r.fields[i].DataTypeOID, r.fields[i].Format, r.values[r.index][i], d); e != nil {
			r.err = e
			return e
		}
	}
	return nil
}
func (r *BufferedRows) Values() ([]any, error) {
	m := pgtype.NewMap()
	out := make([]any, len(r.fields))
	for i, f := range r.fields {
		t, _ := m.TypeForOID(f.DataTypeOID)
		v, e := t.Codec.DecodeValue(m, f.DataTypeOID, r.fields[i].Format, r.values[r.index][i])
		if e != nil {
			return nil, e
		}
		out[i] = v
	}
	return out, nil
}
func (r *BufferedRows) RawValues() [][]byte  { return r.values[r.index] }
func (r *BufferedRows) Conn() *pgx.Conn      { return nil }
func (r *BufferedRows) TypeMap() *pgtype.Map { return pgtype.NewMap() }

// BufferRows releases the database cursor and preserves its wire formats for scanning.
func BufferRows(rows pgx.Rows, err error) (*BufferedRows, [][]any, error) {
	b := &BufferedRows{index: -1, err: err}
	if err != nil {
		if rows != nil {
			rows.Close()
		}
		return b, nil, err
	}
	defer rows.Close()
	b.fields = append([]pgconn.FieldDescription(nil), rows.FieldDescriptions()...)
	var values [][]any
	for rows.Next() {
		v, e := rows.Values()
		if e != nil {
			b.err = e
			return b, nil, e
		}
		values = append(values, v)
		raw := rows.RawValues()
		copyRow := make([][]byte, len(raw))
		for i, v := range raw {
			if v != nil {
				copyRow[i] = append([]byte{}, v...)
			}
		}
		b.values = append(b.values, copyRow)
	}
	b.err = rows.Err()
	b.tag = rows.CommandTag()
	return b, values, b.err
}

// NewBufferedRows wraps owned raw PostgreSQL-format values. JED supplies text format;
// BufferRows preserves the original format when buffering PostgreSQL results.
func NewBufferedRows(fields []pgconn.FieldDescription, values [][][]byte, tag pgconn.CommandTag, err error) *BufferedRows {
	return &BufferedRows{fields: fields, values: values, index: -1, tag: tag, err: err}
}
