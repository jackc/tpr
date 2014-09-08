// Package box stores values, undefined, or null.
package box

import (
	"errors"
	"github.com/jackc/pgx"
	"strconv"
	"time"
)

const (
	Undefined = iota
	Null      = iota
	Full      = iota
)

type Bool struct {
	value  bool
	status byte
}

// NewBool returns a Bool initialized to v
func NewBool(v bool) (box Bool) {
	box.Set(v)
	return box
}

// Set places v in box
func (box *Bool) Set(v bool) {
	box.value = v
	box.status = Full
}

// GetCoerceNil returns the value if the box is full, otherwise it returns nil
func (box *Bool) GetCoerceNil() interface{} {
	if box.status == Full {
		return box.value
	} else {
		return nil
	}
}

// GetCoerceZero returns value if the box is full, otherwise it returns the zero value
func (box *Bool) GetCoerceZero() bool {
	if box.status == Full {
		return box.value
	} else {
		var zero bool
		return zero
	}
}

// SetCoerceZero places v in box if v is not the zero value, otherwise it sets box to zeroStatus
func (box *Bool) SetCoerceZero(v bool, zeroStatus byte) {
	var zero bool

	if v != zero {
		box.Set(v)
	} else {
		box.status = zeroStatus
	}
}

// SetUndefined sets box to Undefined
func (box *Bool) SetUndefined() {
	box.status = Undefined
}

// SetNull sets box to Null
func (box *Bool) SetNull() {
	box.status = Null
}

// MustGet returns the value or panics if box is not full
func (box *Bool) MustGet() bool {
	if box.status != Full {
		panic("called MustGet on a box that was not full")
	}

	return box.value
}

// Get returns the value and present. present is true only if the box is Full and value is valid
func (box *Bool) Get() (bool, bool) {
	if box.status != Full {
		var zeroVal bool
		return zeroVal, false
	}

	return box.value, true
}

// Status returns the box's status
func (box *Bool) Status() byte {
	return box.status
}

type Int16 struct {
	value  int16
	status byte
}

// NewInt16 returns a Int16 initialized to v
func NewInt16(v int16) (box Int16) {
	box.Set(v)
	return box
}

// Set places v in box
func (box *Int16) Set(v int16) {
	box.value = v
	box.status = Full
}

// GetCoerceNil returns the value if the box is full, otherwise it returns nil
func (box *Int16) GetCoerceNil() interface{} {
	if box.status == Full {
		return box.value
	} else {
		return nil
	}
}

// GetCoerceZero returns value if the box is full, otherwise it returns the zero value
func (box *Int16) GetCoerceZero() int16 {
	if box.status == Full {
		return box.value
	} else {
		var zero int16
		return zero
	}
}

// SetCoerceZero places v in box if v is not the zero value, otherwise it sets box to zeroStatus
func (box *Int16) SetCoerceZero(v int16, zeroStatus byte) {
	var zero int16

	if v != zero {
		box.Set(v)
	} else {
		box.status = zeroStatus
	}
}

// SetUndefined sets box to Undefined
func (box *Int16) SetUndefined() {
	box.status = Undefined
}

// SetNull sets box to Null
func (box *Int16) SetNull() {
	box.status = Null
}

// MustGet returns the value or panics if box is not full
func (box *Int16) MustGet() int16 {
	if box.status != Full {
		panic("called MustGet on a box that was not full")
	}

	return box.value
}

// Get returns the value and present. present is true only if the box is Full and value is valid
func (box *Int16) Get() (int16, bool) {
	if box.status != Full {
		var zeroVal int16
		return zeroVal, false
	}

	return box.value, true
}

// Status returns the box's status
func (box *Int16) Status() byte {
	return box.status
}

type Int32 struct {
	value  int32
	status byte
}

// NewInt32 returns a Int32 initialized to v
func NewInt32(v int32) (box Int32) {
	box.Set(v)
	return box
}

// Set places v in box
func (box *Int32) Set(v int32) {
	box.value = v
	box.status = Full
}

// GetCoerceNil returns the value if the box is full, otherwise it returns nil
func (box *Int32) GetCoerceNil() interface{} {
	if box.status == Full {
		return box.value
	} else {
		return nil
	}
}

// GetCoerceZero returns value if the box is full, otherwise it returns the zero value
func (box *Int32) GetCoerceZero() int32 {
	if box.status == Full {
		return box.value
	} else {
		var zero int32
		return zero
	}
}

// SetCoerceZero places v in box if v is not the zero value, otherwise it sets box to zeroStatus
func (box *Int32) SetCoerceZero(v int32, zeroStatus byte) {
	var zero int32

	if v != zero {
		box.Set(v)
	} else {
		box.status = zeroStatus
	}
}

// SetUndefined sets box to Undefined
func (box *Int32) SetUndefined() {
	box.status = Undefined
}

// SetNull sets box to Null
func (box *Int32) SetNull() {
	box.status = Null
}

// MustGet returns the value or panics if box is not full
func (box *Int32) MustGet() int32 {
	if box.status != Full {
		panic("called MustGet on a box that was not full")
	}

	return box.value
}

// Get returns the value and present. present is true only if the box is Full and value is valid
func (box *Int32) Get() (int32, bool) {
	if box.status != Full {
		var zeroVal int32
		return zeroVal, false
	}

	return box.value, true
}

// Status returns the box's status
func (box *Int32) Status() byte {
	return box.status
}

type Int64 struct {
	value  int64
	status byte
}

// NewInt64 returns a Int64 initialized to v
func NewInt64(v int64) (box Int64) {
	box.Set(v)
	return box
}

// Set places v in box
func (box *Int64) Set(v int64) {
	box.value = v
	box.status = Full
}

// GetCoerceNil returns the value if the box is full, otherwise it returns nil
func (box *Int64) GetCoerceNil() interface{} {
	if box.status == Full {
		return box.value
	} else {
		return nil
	}
}

// GetCoerceZero returns value if the box is full, otherwise it returns the zero value
func (box *Int64) GetCoerceZero() int64 {
	if box.status == Full {
		return box.value
	} else {
		var zero int64
		return zero
	}
}

// SetCoerceZero places v in box if v is not the zero value, otherwise it sets box to zeroStatus
func (box *Int64) SetCoerceZero(v int64, zeroStatus byte) {
	var zero int64

	if v != zero {
		box.Set(v)
	} else {
		box.status = zeroStatus
	}
}

// SetUndefined sets box to Undefined
func (box *Int64) SetUndefined() {
	box.status = Undefined
}

// SetNull sets box to Null
func (box *Int64) SetNull() {
	box.status = Null
}

// MustGet returns the value or panics if box is not full
func (box *Int64) MustGet() int64 {
	if box.status != Full {
		panic("called MustGet on a box that was not full")
	}

	return box.value
}

// Get returns the value and present. present is true only if the box is Full and value is valid
func (box *Int64) Get() (int64, bool) {
	if box.status != Full {
		var zeroVal int64
		return zeroVal, false
	}

	return box.value, true
}

// Status returns the box's status
func (box *Int64) Status() byte {
	return box.status
}

type String struct {
	value  string
	status byte
}

// NewString returns a String initialized to v
func NewString(v string) (box String) {
	box.Set(v)
	return box
}

// Set places v in box
func (box *String) Set(v string) {
	box.value = v
	box.status = Full
}

// GetCoerceNil returns the value if the box is full, otherwise it returns nil
func (box *String) GetCoerceNil() interface{} {
	if box.status == Full {
		return box.value
	} else {
		return nil
	}
}

// GetCoerceZero returns value if the box is full, otherwise it returns the zero value
func (box *String) GetCoerceZero() string {
	if box.status == Full {
		return box.value
	} else {
		var zero string
		return zero
	}
}

// SetCoerceZero places v in box if v is not the zero value, otherwise it sets box to zeroStatus
func (box *String) SetCoerceZero(v string, zeroStatus byte) {
	var zero string

	if v != zero {
		box.Set(v)
	} else {
		box.status = zeroStatus
	}
}

// SetUndefined sets box to Undefined
func (box *String) SetUndefined() {
	box.status = Undefined
}

// SetNull sets box to Null
func (box *String) SetNull() {
	box.status = Null
}

// MustGet returns the value or panics if box is not full
func (box *String) MustGet() string {
	if box.status != Full {
		panic("called MustGet on a box that was not full")
	}

	return box.value
}

// Get returns the value and present. present is true only if the box is Full and value is valid
func (box *String) Get() (string, bool) {
	if box.status != Full {
		var zeroVal string
		return zeroVal, false
	}

	return box.value, true
}

// Status returns the box's status
func (box *String) Status() byte {
	return box.status
}

type Time struct {
	value  time.Time
	status byte
}

// NewTime returns a Time initialized to v
func NewTime(v time.Time) (box Time) {
	box.Set(v)
	return box
}

// Set places v in box
func (box *Time) Set(v time.Time) {
	box.value = v
	box.status = Full
}

// GetCoerceNil returns the value if the box is full, otherwise it returns nil
func (box *Time) GetCoerceNil() interface{} {
	if box.status == Full {
		return box.value
	} else {
		return nil
	}
}

// GetCoerceZero returns value if the box is full, otherwise it returns the zero value
func (box *Time) GetCoerceZero() time.Time {
	if box.status == Full {
		return box.value
	} else {
		var zero time.Time
		return zero
	}
}

// SetCoerceZero places v in box if v is not the zero value, otherwise it sets box to zeroStatus
func (box *Time) SetCoerceZero(v time.Time, zeroStatus byte) {
	var zero time.Time

	if v != zero {
		box.Set(v)
	} else {
		box.status = zeroStatus
	}
}

// SetUndefined sets box to Undefined
func (box *Time) SetUndefined() {
	box.status = Undefined
}

// SetNull sets box to Null
func (box *Time) SetNull() {
	box.status = Null
}

// MustGet returns the value or panics if box is not full
func (box *Time) MustGet() time.Time {
	if box.status != Full {
		panic("called MustGet on a box that was not full")
	}

	return box.value
}

// Get returns the value and present. present is true only if the box is Full and value is valid
func (box *Time) Get() (time.Time, bool) {
	if box.status != Full {
		var zeroVal time.Time
		return zeroVal, false
	}

	return box.value, true
}

// Status returns the box's status
func (box *Time) Status() byte {
	return box.status
}

func (box Bool) MarshalJSON() ([]byte, error) {
	if box.status != Full {
		return []byte("null"), nil
	}
	if box.value {
		return []byte("true"), nil
	}
	return []byte("false"), nil
}

func (box Int16) MarshalJSON() ([]byte, error) {
	if box.status != Full {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(int64(box.value), 10)), nil
}

func (box Int32) MarshalJSON() ([]byte, error) {
	if box.status != Full {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(int64(box.value), 10)), nil
}

func (box Int64) MarshalJSON() ([]byte, error) {
	if box.status != Full {
		return []byte("null"), nil
	}
	return []byte(strconv.FormatInt(int64(box.value), 10)), nil
}

func (box String) MarshalJSON() ([]byte, error) {
	if box.status != Full {
		return []byte("null"), nil
	}
	return []byte(`"` + box.value + `"`), nil
}

func (box *Bool) Scan(r *pgx.ValueReader) error {
	var nv pgx.NullBool
	err := nv.Scan(r)
	if err != nil {
		return err
	}

	box.value = nv.Bool
	if nv.Valid {
		box.status = Full
	} else {
		box.status = Null
	}

	return nil
}

func (box Bool) FormatCode() int16 {
	var nv pgx.NullBool
	return nv.FormatCode()
}

func (box Bool) Encode(w *pgx.WriteBuf, oid pgx.Oid) error {
	var nv pgx.NullBool
	nv.Bool = box.value

	switch box.status {
	case Full:
		nv.Valid = true
	case Null:
		nv.Valid = false
	case Undefined:
		return errors.New("cannot encode undefined box")
	}

	return nv.Encode(w, oid)
}

func (box *Int16) Scan(r *pgx.ValueReader) error {
	var nv pgx.NullInt16
	err := nv.Scan(r)
	if err != nil {
		return err
	}

	box.value = nv.Int16
	if nv.Valid {
		box.status = Full
	} else {
		box.status = Null
	}

	return nil
}

func (box Int16) FormatCode() int16 {
	var nv pgx.NullInt16
	return nv.FormatCode()
}

func (box Int16) Encode(w *pgx.WriteBuf, oid pgx.Oid) error {
	var nv pgx.NullInt16
	nv.Int16 = box.value

	switch box.status {
	case Full:
		nv.Valid = true
	case Null:
		nv.Valid = false
	case Undefined:
		return errors.New("cannot encode undefined box")
	}

	return nv.Encode(w, oid)
}

func (box *Int32) Scan(r *pgx.ValueReader) error {
	var nv pgx.NullInt32
	err := nv.Scan(r)
	if err != nil {
		return err
	}

	box.value = nv.Int32
	if nv.Valid {
		box.status = Full
	} else {
		box.status = Null
	}

	return nil
}

func (box Int32) FormatCode() int16 {
	var nv pgx.NullInt32
	return nv.FormatCode()
}

func (box Int32) Encode(w *pgx.WriteBuf, oid pgx.Oid) error {
	var nv pgx.NullInt32
	nv.Int32 = box.value

	switch box.status {
	case Full:
		nv.Valid = true
	case Null:
		nv.Valid = false
	case Undefined:
		return errors.New("cannot encode undefined box")
	}

	return nv.Encode(w, oid)
}

func (box *Int64) Scan(r *pgx.ValueReader) error {
	var nv pgx.NullInt64
	err := nv.Scan(r)
	if err != nil {
		return err
	}

	box.value = nv.Int64
	if nv.Valid {
		box.status = Full
	} else {
		box.status = Null
	}

	return nil
}

func (box Int64) FormatCode() int16 {
	var nv pgx.NullInt64
	return nv.FormatCode()
}

func (box Int64) Encode(w *pgx.WriteBuf, oid pgx.Oid) error {
	var nv pgx.NullInt64
	nv.Int64 = box.value

	switch box.status {
	case Full:
		nv.Valid = true
	case Null:
		nv.Valid = false
	case Undefined:
		return errors.New("cannot encode undefined box")
	}

	return nv.Encode(w, oid)
}

func (box *String) Scan(r *pgx.ValueReader) error {
	var nv pgx.NullString
	err := nv.Scan(r)
	if err != nil {
		return err
	}

	box.value = nv.String
	if nv.Valid {
		box.status = Full
	} else {
		box.status = Null
	}

	return nil
}

func (box String) FormatCode() int16 {
	var nv pgx.NullString
	return nv.FormatCode()
}

func (box String) Encode(w *pgx.WriteBuf, oid pgx.Oid) error {
	var nv pgx.NullString
	nv.String = box.value

	switch box.status {
	case Full:
		nv.Valid = true
	case Null:
		nv.Valid = false
	case Undefined:
		return errors.New("cannot encode undefined box")
	}

	return nv.Encode(w, oid)
}

func (box *Time) Scan(r *pgx.ValueReader) error {
	var nv pgx.NullTime
	err := nv.Scan(r)
	if err != nil {
		return err
	}

	box.value = nv.Time
	if nv.Valid {
		box.status = Full
	} else {
		box.status = Null
	}

	return nil
}

func (box Time) FormatCode() int16 {
	var nv pgx.NullTime
	return nv.FormatCode()
}

func (box Time) Encode(w *pgx.WriteBuf, oid pgx.Oid) error {
	var nv pgx.NullTime
	nv.Time = box.value

	switch box.status {
	case Full:
		nv.Valid = true
	case Null:
		nv.Valid = false
	case Undefined:
		return errors.New("cannot encode undefined box")
	}

	return nv.Encode(w, oid)
}
