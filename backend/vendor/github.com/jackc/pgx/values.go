package pgx

import (
	"bytes"
	"database/sql/driver"
	"fmt"
	"math"
	"reflect"
	"time"

	"github.com/jackc/pgx/pgtype"
)

// PostgreSQL format codes
const (
	TextFormatCode   = 0
	BinaryFormatCode = 1
)

// SerializationError occurs on failure to encode or decode a value
type SerializationError string

func (e SerializationError) Error() string {
	return string(e)
}

func convertSimpleArgument(ci *pgtype.ConnInfo, arg interface{}) (interface{}, error) {
	if arg == nil {
		return nil, nil
	}

	switch arg := arg.(type) {
	case driver.Valuer:
		return arg.Value()
	case pgtype.TextEncoder:
		buf := &bytes.Buffer{}
		null, err := arg.EncodeText(ci, buf)
		if err != nil {
			return nil, err
		}
		if null {
			return nil, nil
		}
		return buf.String(), nil
	case int64:
		return arg, nil
	case float64:
		return arg, nil
	case bool:
		return arg, nil
	case time.Time:
		return arg, nil
	case string:
		return arg, nil
	case []byte:
		return arg, nil
	case int8:
		return int64(arg), nil
	case int16:
		return int64(arg), nil
	case int32:
		return int64(arg), nil
	case int:
		return int64(arg), nil
	case uint8:
		return int64(arg), nil
	case uint16:
		return int64(arg), nil
	case uint32:
		return int64(arg), nil
	case uint64:
		if arg > math.MaxInt64 {
			return nil, fmt.Errorf("arg too big for int64: %v", arg)
		}
		return int64(arg), nil
	case uint:
		if arg > math.MaxInt64 {
			return nil, fmt.Errorf("arg too big for int64: %v", arg)
		}
		return int64(arg), nil
	case float32:
		return float64(arg), nil
	}

	refVal := reflect.ValueOf(arg)

	if refVal.Kind() == reflect.Ptr {
		if refVal.IsNil() {
			return nil, nil
		}
		arg = refVal.Elem().Interface()
		return convertSimpleArgument(ci, arg)
	}

	if strippedArg, ok := stripNamedType(&refVal); ok {
		return convertSimpleArgument(ci, strippedArg)
	}
	return nil, SerializationError(fmt.Sprintf("Cannot encode %T in simple protocol - %T must implement driver.Valuer, pgtype.TextEncoder, or be a native type", arg, arg))
}

func encodePreparedStatementArgument(wbuf *WriteBuf, oid pgtype.Oid, arg interface{}) error {
	if arg == nil {
		wbuf.WriteInt32(-1)
		return nil
	}

	switch arg := arg.(type) {
	case pgtype.BinaryEncoder:
		buf := &bytes.Buffer{}
		null, err := arg.EncodeBinary(wbuf.conn.ConnInfo, buf)
		if err != nil {
			return err
		}
		if null {
			wbuf.WriteInt32(-1)
		} else {
			wbuf.WriteInt32(int32(buf.Len()))
			wbuf.WriteBytes(buf.Bytes())
		}
		return nil
	case pgtype.TextEncoder:
		buf := &bytes.Buffer{}
		null, err := arg.EncodeText(wbuf.conn.ConnInfo, buf)
		if err != nil {
			return err
		}
		if null {
			wbuf.WriteInt32(-1)
		} else {
			wbuf.WriteInt32(int32(buf.Len()))
			wbuf.WriteBytes(buf.Bytes())
		}
		return nil
	case driver.Valuer:
		v, err := arg.Value()
		if err != nil {
			return err
		}
		return encodePreparedStatementArgument(wbuf, oid, v)
	case string:
		wbuf.WriteInt32(int32(len(arg)))
		wbuf.WriteBytes([]byte(arg))
		return nil
	}

	refVal := reflect.ValueOf(arg)

	if refVal.Kind() == reflect.Ptr {
		if refVal.IsNil() {
			wbuf.WriteInt32(-1)
			return nil
		}
		arg = refVal.Elem().Interface()
		return encodePreparedStatementArgument(wbuf, oid, arg)
	}

	if dt, ok := wbuf.conn.ConnInfo.DataTypeForOid(oid); ok {
		value := dt.Value
		err := value.Set(arg)
		if err != nil {
			return err
		}

		buf := &bytes.Buffer{}
		null, err := value.(pgtype.BinaryEncoder).EncodeBinary(wbuf.conn.ConnInfo, buf)
		if err != nil {
			return err
		}
		if null {
			wbuf.WriteInt32(-1)
		} else {
			wbuf.WriteInt32(int32(buf.Len()))
			wbuf.WriteBytes(buf.Bytes())
		}
		return nil
	}

	if strippedArg, ok := stripNamedType(&refVal); ok {
		return encodePreparedStatementArgument(wbuf, oid, strippedArg)
	}
	return SerializationError(fmt.Sprintf("Cannot encode %T into oid %v - %T must implement Encoder or be converted to a string", arg, oid, arg))
}

// chooseParameterFormatCode determines the correct format code for an
// argument to a prepared statement. It defaults to TextFormatCode if no
// determination can be made.
func chooseParameterFormatCode(ci *pgtype.ConnInfo, oid pgtype.Oid, arg interface{}) int16 {
	switch arg.(type) {
	case pgtype.BinaryEncoder:
		return BinaryFormatCode
	case string, *string, pgtype.TextEncoder:
		return TextFormatCode
	}

	if dt, ok := ci.DataTypeForOid(oid); ok {
		if _, ok := dt.Value.(pgtype.BinaryEncoder); ok {
			if arg, ok := arg.(driver.Valuer); ok {
				if err := dt.Value.Set(arg); err != nil {
					if value, err := arg.Value(); err == nil {
						if _, ok := value.(string); ok {
							return TextFormatCode
						}
					}
				}
			}

			return BinaryFormatCode
		}
	}

	return TextFormatCode
}

func stripNamedType(val *reflect.Value) (interface{}, bool) {
	switch val.Kind() {
	case reflect.Int:
		convVal := int(val.Int())
		return convVal, reflect.TypeOf(convVal) != val.Type()
	case reflect.Int8:
		convVal := int8(val.Int())
		return convVal, reflect.TypeOf(convVal) != val.Type()
	case reflect.Int16:
		convVal := int16(val.Int())
		return convVal, reflect.TypeOf(convVal) != val.Type()
	case reflect.Int32:
		convVal := int32(val.Int())
		return convVal, reflect.TypeOf(convVal) != val.Type()
	case reflect.Int64:
		convVal := int64(val.Int())
		return convVal, reflect.TypeOf(convVal) != val.Type()
	case reflect.Uint:
		convVal := uint(val.Uint())
		return convVal, reflect.TypeOf(convVal) != val.Type()
	case reflect.Uint8:
		convVal := uint8(val.Uint())
		return convVal, reflect.TypeOf(convVal) != val.Type()
	case reflect.Uint16:
		convVal := uint16(val.Uint())
		return convVal, reflect.TypeOf(convVal) != val.Type()
	case reflect.Uint32:
		convVal := uint32(val.Uint())
		return convVal, reflect.TypeOf(convVal) != val.Type()
	case reflect.Uint64:
		convVal := uint64(val.Uint())
		return convVal, reflect.TypeOf(convVal) != val.Type()
	case reflect.String:
		convVal := val.String()
		return convVal, reflect.TypeOf(convVal) != val.Type()
	}

	return nil, false
}
