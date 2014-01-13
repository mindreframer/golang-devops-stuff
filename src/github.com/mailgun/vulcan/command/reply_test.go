package command

import (
	"encoding/json"
	. "launchpad.net/gocheck"
)

type ReplySuite struct{}

var _ = Suite(&ReplySuite{})

func (s *ReplySuite) TestReplySuccess(c *C) {
	commands := []struct {
		Expected *Reply
		Parse    string
	}{
		{
			Parse: `{"code": 500, "body": "access denied"}`,
			Expected: &Reply{
				Code: 500,
				Body: "access denied",
			},
		},
		{
			Parse: `{"code": 405, "body": {"error": "some error"}}`,
			Expected: &Reply{
				Code: 405,
				Body: map[string]interface{}{"error": "some error"},
			},
		},
	}

	for _, cmd := range commands {
		var value interface{}
		err := json.Unmarshal([]byte(cmd.Parse), &value)
		c.Assert(err, IsNil)
		parsed, err := NewCommandFromObj(value)
		c.Assert(err, IsNil)
		c.Assert(parsed, DeepEquals, cmd.Expected)
	}
}

func (s *ReplySuite) TestWrongReplies(c *C) {
	replies := []string{
		//missing code
		`{"body": "access denied"}`,
		//missing body
		`{"code": 200}`,
		// wrong code (not integer)
		`{"code": 50.2, "body": "access denied"}`,
		// wrong code (not integer)
		`{"code": "some code", "body": "access denied"}`,
		// wrong code (negative)
		`{"code": -100, "body": "access denied"}`,
	}
	for _, str := range replies {
		var value map[string]interface{}
		err := json.Unmarshal([]byte(str), &value)
		c.Assert(err, IsNil)

		_, err = NewReplyFromDict(value)
		c.Assert(err, Not(IsNil))
	}
}
