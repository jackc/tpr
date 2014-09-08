package box_test

import (
	"encoding/json"
	"github.com/jackc/tpr/backend/box"
	. "launchpad.net/gocheck"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func (s *MySuite) TestZeroValueTimeIsUndefined(c *C) {
	var b box.Time
	c.Check(b.Status(), Equals, byte(box.Undefined))
}

func (s *MySuite) TestNewTime(c *C) {
	val := time.Now()
	b := box.NewTime(val)

	val2, present := b.Get()
	c.Check(val2, Equals, val)
	c.Check(present, Equals, true)
}

func (s *MySuite) TestSetAndGet(c *C) {
	var b box.Time
	val := time.Now()

	b.Set(val)

	val2, present := b.Get()
	c.Check(val2, Equals, val)
	c.Check(present, Equals, true)

	c.Check(b.MustGet(), Equals, val)

	b.SetUndefined()
	_, present = b.Get()
	c.Check(present, Equals, false)

	b.SetNull()
	_, present = b.Get()
	c.Check(present, Equals, false)

}

func (s *MySuite) TestGetCoerceNil(c *C) {
	var b box.Time

	b.SetUndefined()
	c.Check(b.GetCoerceNil(), Equals, nil)

	b.SetNull()
	c.Check(b.GetCoerceNil(), Equals, nil)

	val := time.Now()
	b.Set(val)
	c.Check(b.GetCoerceNil(), Equals, val)
}

func (s *MySuite) TestGetCoerceZero(c *C) {
	var b box.Time
	var zero time.Time

	b.SetUndefined()
	c.Check(b.GetCoerceZero(), Equals, zero)

	b.SetNull()
	c.Check(b.GetCoerceZero(), Equals, zero)

	val := time.Now()
	b.Set(val)
	c.Check(b.GetCoerceNil(), Equals, val)
}

func (s *MySuite) TestSetCoerceZero(c *C) {
	var b box.Time
	var zero time.Time

	b.SetCoerceZero(zero, box.Null)
	c.Check(b.Status(), Equals, byte(box.Null))
}

func (s *MySuite) TestMustGetPanicsWhenNotFull(c *C) {
	var b box.Time

	b.SetUndefined()
	c.Check(b.MustGet, Panics, "called MustGet on a box that was not full")

	b.SetNull()
	c.Check(b.MustGet, Panics, "called MustGet on a box that was not full")
}

func (s *MySuite) TestJSONMarshal(c *C) {
	var tests = []struct {
		val      json.Marshaler
		expected string
	}{
		{box.NewBool(true), "true"},
		{box.NewBool(false), "false"},
		{box.Bool{}, "null"},
		{box.NewInt16(1), "1"},
		{box.NewInt16(-1), "-1"},
		{box.Int16{}, "null"},
		{box.NewInt32(1), "1"},
		{box.NewInt32(-1), "-1"},
		{box.Int32{}, "null"},
		{box.NewInt64(1), "1"},
		{box.NewInt64(-1), "-1"},
		{box.Int64{}, "null"},
		{box.NewString("foo"), `"foo"`},
		{box.String{}, "null"},
	}

	for i, t := range tests {
		jsonBytes, err := t.val.MarshalJSON()
		if err != nil {
			c.Errorf("%d. MarshalJSON unexpectedly returned an error: %s", i, err)
			continue
		}
		actual := string(jsonBytes)
		if actual != t.expected {
			c.Errorf(`%d. Expected MarshalJSON to return "%s", but it returned "%s"`, i, t.expected, actual)
		}
	}
}
