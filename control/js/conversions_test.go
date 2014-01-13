package js

import (
	. "launchpad.net/gocheck"
	"net/url"
)

type ConversionsSuite struct{}

var _ = Suite(&ConversionsSuite{})

func (s *ConversionsSuite) TestToString(c *C) {
	val, err := toString("Hello")
	c.Assert(err, Equals, nil)
	c.Assert(val, Equals, "Hello")

	_, err = toString(map[string]interface{}{})
	c.Assert(err, Not(Equals), nil)
}

func (s *ConversionsSuite) TestToStringArraySuccess(c *C) {
	commands := []struct {
		In       interface{}
		Expected []string
	}{
		{
			In:       []string{"a", "b"},
			Expected: []string{"a", "b"},
		},
		{
			In:       "a",
			Expected: []string{"a"},
		},
		{
			In:       []interface{}{"a", "b"},
			Expected: []string{"a", "b"},
		},
	}

	for _, in := range commands {
		out, err := toStringArray(in.In)
		c.Assert(err, Equals, nil)
		c.Assert(out, DeepEquals, in.Expected)
	}
}

func (s *ConversionsSuite) TestToStringArrayFailure(c *C) {
	commands := []interface{}{
		// Non string types
		1,
		[]byte("Nope"),
		map[string]interface{}{},
	}

	for _, in := range commands {
		_, err := toStringArray(in)
		c.Assert(err, Not(Equals), nil)
	}
}

func (s *ConversionsSuite) TestToMultiDictSuccess(c *C) {
	commands := []struct {
		In       interface{}
		Expected map[string][]string
	}{
		{
			In:       map[string][]string{"a": []string{"b"}},
			Expected: map[string][]string{"a": []string{"b"}},
		},
		{
			In:       url.Values{"a": []string{"b"}},
			Expected: map[string][]string{"a": []string{"b"}},
		},
		{
			In:       map[string]interface{}{"a": "b"},
			Expected: map[string][]string{"a": []string{"b"}},
		},
		{
			In:       map[string]interface{}{"a": []interface{}{"b"}},
			Expected: map[string][]string{"a": []string{"b"}},
		},
		{
			In:       map[string]interface{}{"a": []string{"b"}},
			Expected: map[string][]string{"a": []string{"b"}},
		},
	}

	for _, in := range commands {
		out, err := toMultiDict(in.In)
		c.Assert(err, Equals, nil)
		c.Assert(out, DeepEquals, in.Expected)
	}
}

func (s *ConversionsSuite) TestToMultiDictFailure(c *C) {
	commands := []interface{}{
		// Broken types
		1,
		[]byte("Nope"),
		map[string]interface{}{"a": 1},
	}

	for _, in := range commands {
		_, err := toMultiDict(in)
		c.Assert(err, Not(Equals), nil)
	}
}
