package js

import (
	"github.com/mailgun/vulcan/netutils"
	. "launchpad.net/gocheck"
	"net/http"
	"net/url"
)

type RequestSuite struct{}

var _ = Suite(&RequestSuite{})

func (s *RequestSuite) TestRequestToJs(c *C) {
	commands := []struct {
		Request  *http.Request
		Expected map[string]interface{}
	}{
		{
			Request: NewTestRequest("GET", "http://localhost", nil),
			Expected: map[string]interface{}{
				"auth": map[string]interface{}{
					"username": "",
					"password": "",
				},
				"url":      "http://localhost",
				"path":     "",
				"query":    url.Values{},
				"protocol": "HTTP/1.1",
				"method":   "GET",
				"length":   int64(0),
				"headers":  http.Header{},
			},
		},
		{
			Request: NewTestRequest(
				"GET", "http://localhost/path?a=b", &netutils.BasicAuth{"user", "pass"}),
			Expected: map[string]interface{}{
				"auth": map[string]interface{}{
					"username": "user",
					"password": "pass",
				},
				"url":      "http://localhost/path?a=b",
				"path":     "/path",
				"query":    url.Values{"a": []string{"b"}},
				"protocol": "HTTP/1.1",
				"method":   "GET",
				"length":   int64(0),
				"headers":  http.Header{"Authorization": []string{"Basic dXNlcjpwYXNz"}},
			},
		},
	}

	for _, in := range commands {
		r, err := requestToJs(in.Request)
		c.Assert(err, Equals, nil)
		for key, val := range r {
			c.Assert(val, DeepEquals, in.Expected[key])
		}
	}
}
