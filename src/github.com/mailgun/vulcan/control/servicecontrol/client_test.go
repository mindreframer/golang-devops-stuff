package servicecontrol

import (
	. "launchpad.net/gocheck"
	"net/http"
	"testing"
)

func TestClient(t *testing.T) { TestingT(t) }

type ClientSuite struct{}

var _ = Suite(&ClientSuite{})

func (s *ClientSuite) TestFromHttpSuccess(c *C) {
	requests := []struct {
		In  http.Request
		Out ControlRequest
	}{
		{
			http.Request{
				Method: "GET",
				Header: map[string][]string{
					"Authorization": []string{"Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ=="},
				}},
			ControlRequest{Method: "GET"},
		},
	}
	for _, r := range requests {
		_, err := controlRequestFromHttp(&r.In)
		c.Assert(err, IsNil)
	}
}

func (s *ClientSuite) TestFromHttpFail(c *C) {
	requests := []http.Request{
		http.Request{
			Method: "GET",
			Header: map[string][]string{
				"Authorization": []string{"Broken auth"},
			}},
	}
	for _, r := range requests {
		_, err := controlRequestFromHttp(&r)
		c.Assert(err, NotNil)
	}
}
