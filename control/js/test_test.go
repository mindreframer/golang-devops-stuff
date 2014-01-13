package js

import (
	"github.com/mailgun/vulcan/command"
	"github.com/mailgun/vulcan/netutils"
	. "launchpad.net/gocheck"
	"net/http"
	"testing"
)

func TestCommand(t *testing.T) { TestingT(t) }

func NewTestRequest(method string, url string, auth *netutils.BasicAuth) *http.Request {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		panic(err)
	}
	if auth != nil {
		req.SetBasicAuth(auth.Username, auth.Password)
	}
	return req
}

func NewTestUpstream(in string) *command.Upstream {
	u, err := command.NewUpstreamFromString(in)
	if err != nil {
		panic(err)
	}
	return u
}
