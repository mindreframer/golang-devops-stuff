package command

import (
	"encoding/json"
	"github.com/mailgun/vulcan/metrics"
	. "launchpad.net/gocheck"
	"net/http"
	"time"
)

type ForwardSuite struct{}

var _ = Suite(&ForwardSuite{})

func (s *ForwardSuite) TestForwardSuccess(c *C) {
	commands := []struct {
		Expected *Forward
		Parse    string
	}{
		{
			Parse: `{"upstreams": ["http://localhost:5000", "http://localhost:5001"]}`,
			Expected: &Forward{
				Upstreams: []*Upstream{
					&Upstream{
						Id:      "http://localhost:5000",
						Scheme:  "http",
						Port:    5000,
						Host:    "localhost",
						Metrics: metrics.GetUpstreamMetrics("http_localhost_5000"),
					},
					&Upstream{
						Id:      "http://localhost:5001",
						Scheme:  "http",
						Port:    5001,
						Host:    "localhost",
						Metrics: metrics.GetUpstreamMetrics("http_localhost_5001"),
					},
				},
			},
		},
		{
			Parse: `{"rates": {"$request.ip": "1 req/second"}, "upstreams": ["http://localhost:5000", "http://localhost:5001"]}`,
			Expected: &Forward{
				Rates: map[string][]*Rate{
					"$request.ip": []*Rate{&Rate{Units: 1, Period: time.Second}},
				},
				Upstreams: []*Upstream{
					&Upstream{
						Id:      "http://localhost:5000",
						Scheme:  "http",
						Port:    5000,
						Host:    "localhost",
						Metrics: metrics.GetUpstreamMetrics("http_localhost_5000"),
					},
					&Upstream{
						Id:      "http://localhost:5001",
						Scheme:  "http",
						Port:    5001,
						Host:    "localhost",
						Metrics: metrics.GetUpstreamMetrics("http_localhost_5001"),
					},
				},
			},
		},
		{
			Parse: `{
                  "failover": {"active": true, "codes": [301, 302]},
                  "rates": {
                     "$request.ip": [
                         "1 req/second",
                         {"KB": 8, "period": "hour"}
                  ]},
                  "upstreams": [
                       "http://localhost:5000",
                        {
                           "scheme": "http",
                           "host": "localhost",
                           "port": 5001
                        }
                  ],
                "add_headers": {"N": "v1"},
                "remove_headers": ["M"],
                "rewrite_path": "/new/path"
            }`,
			Expected: &Forward{
				Failover: &Failover{Active: true, Codes: []int{301, 302}},
				Rates: map[string][]*Rate{
					"$request.ip": []*Rate{
						&Rate{Units: 1, Period: time.Second},
						&Rate{Units: 8, UnitType: UnitTypeKilobytes, Period: time.Hour},
					},
				},
				AddHeaders:    http.Header{"N": []string{"v1"}},
				RemoveHeaders: []string{"M"},
				RewritePath:   "/new/path",
				Upstreams: []*Upstream{
					&Upstream{
						Id:      "http://localhost:5000",
						Scheme:  "http",
						Port:    5000,
						Host:    "localhost",
						Metrics: metrics.GetUpstreamMetrics("http_localhost_5000"),
					},
					&Upstream{
						Id:      "http://localhost:5001",
						Scheme:  "http",
						Port:    5001,
						Host:    "localhost",
						Metrics: metrics.GetUpstreamMetrics("http_localhost_5001"),
					},
				},
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
