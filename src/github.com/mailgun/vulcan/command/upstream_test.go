package command

import (
	"encoding/json"
	"github.com/mailgun/vulcan/metrics"
	. "launchpad.net/gocheck"
)

type UpstreamSuite struct{}

var _ = Suite(&UpstreamSuite{})

func (s *UpstreamSuite) TestNewUpstream(c *C) {
	u, err := NewUpstream(
		"http",
		"localhost",
		5000)
	c.Assert(err, IsNil)
	expected := Upstream{
		Id:      "http://localhost:5000",
		Scheme:  "http",
		Host:    "localhost",
		Port:    5000,
		Metrics: metrics.GetUpstreamMetrics("http_localhost_5000"),
	}
	c.Assert(*u, DeepEquals, expected)
}

func (s *UpstreamSuite) TestNewUpstreamNoPort(c *C) {
	u, err := NewUpstreamFromString("http://localhost")
	c.Assert(err, IsNil)
	expected := Upstream{
		Id:      "http://localhost:80",
		Scheme:  "http",
		Host:    "localhost",
		Port:    80,
		Metrics: metrics.GetUpstreamMetrics("http_localhost_80"),
	}
	c.Assert(*u, DeepEquals, expected)
}

func (s *UpstreamSuite) TestUpstreamFromObj(c *C) {
	upstreams := []struct {
		Expected Upstream
		Parse    string
	}{
		{
			Parse: `"http://google.com:5000"`,
			Expected: Upstream{
				Id:      "http://google.com:5000",
				Scheme:  "http",
				Host:    "google.com",
				Port:    5000,
				Metrics: metrics.GetUpstreamMetrics("http_google_com_5000"),
			},
		},
		{
			Parse: `"http://google.com:5000/"`,
			Expected: Upstream{
				Id:      "http://google.com:5000",
				Scheme:  "http",
				Host:    "google.com",
				Port:    5000,
				Metrics: metrics.GetUpstreamMetrics("http_google_com_5000"),
			},
		},
		{
			Parse: `{"scheme": "http", "host": "localhost", "port": 3000}`,
			Expected: Upstream{
				Id:      "http://localhost:3000",
				Scheme:  "http",
				Host:    "localhost",
				Port:    3000,
				Metrics: metrics.GetUpstreamMetrics("http_localhost_3000"),
			},
		},
		{
			Parse: `{"scheme": "https", "host": "localhost", "port": 4000}`,
			Expected: Upstream{
				Id:      "https://localhost:4000",
				Scheme:  "https",
				Host:    "localhost",
				Port:    4000,
				Metrics: metrics.GetUpstreamMetrics("https_localhost_4000"),
			},
		},
	}

	for _, u := range upstreams {
		var value interface{}
		err := json.Unmarshal([]byte(u.Parse), &value)
		c.Assert(err, IsNil)
		parsed, err := NewUpstreamFromObj(value)
		c.Assert(err, IsNil)
		c.Assert(u.Expected, DeepEquals, *parsed)
	}
}
