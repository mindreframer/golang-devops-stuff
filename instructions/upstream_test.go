package instructions

import (
	. "launchpad.net/gocheck"
	"testing"
)

func TestUpstream(t *testing.T) { TestingT(t) }

type UpstreamSuite struct{}

var _ = Suite(&UpstreamSuite{})

// Make sure that upstream ids do not depend on the path
func (s *UpstreamSuite) TestUpstreamIds(c *C) {
	dataSet := []struct {
		url        string
		expectedId string
	}{
		{
			url:        "http://google.com:1245/hello",
			expectedId: "http://google.com:1245",
		},
		{
			url:        "http://google.com",
			expectedId: "http://google.com",
		},
		{
			url:        "http://google.com/what?nothing=true",
			expectedId: "http://google.com",
		},
	}
	for _, s := range dataSet {
		u, err := NewUpstream(s.url, []*Rate{}, map[string][]string{})
		c.Assert(err, IsNil)
		c.Assert(u.Id(), Equals, s.expectedId)
	}
}
