package js

import (
	"encoding/json"
	"fmt"
	"github.com/mailgun/vulcan/command"
	. "launchpad.net/gocheck"
)

type ErrorSuite struct{}

var _ = Suite(&ErrorSuite{})

func (s *ErrorSuite) TestErrorToJs(c *C) {
	commands := []struct {
		Error    error
		Expected map[string]interface{}
	}{
		{
			Error: fmt.Errorf("Any random error"),
			Expected: map[string]interface{}{
				"type": "internal",
				"code": 500,
				"body": map[string]interface{}{
					"error": "Internal Server Error",
				},
			},
		},
		{
			Error: &command.RetryError{Seconds: 10},
			Expected: map[string]interface{}{
				"type":          "retry",
				"retry_seconds": 10,
				"code":          429,
				"body": map[string]interface{}{
					"error":         "Too Many Requests",
					"retry_seconds": 10,
				},
			},
		},
		{
			Error: &command.AllUpstreamsDownError{},
			Expected: map[string]interface{}{
				"type": "all_upstreams_down",
				"code": 502,
				"body": map[string]interface{}{
					"error": "Bad Gateway",
				},
			},
		},
	}
	for _, in := range commands {
		out := errorToJs(in.Error)
		c.Assert(out, DeepEquals, in.Expected)

		netErr, err := errorFromJs(out)
		c.Assert(err, Equals, nil)
		c.Assert(netErr.StatusCode, Equals, in.Expected["code"])
		bytes, err := json.Marshal(in.Expected["body"])
		c.Assert(err, Equals, nil)
		c.Assert(bytes, DeepEquals, netErr.Body)
	}
}
