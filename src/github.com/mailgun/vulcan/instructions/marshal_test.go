package instructions

import (
	. "launchpad.net/gocheck"
	"net/url"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) }

type MarshalSuite struct{}

var _ = Suite(&MarshalSuite{})

func (s *MarshalSuite) TestUnmarshalSuccessBig(c *C) {
	objects := []struct {
		Bytes    []byte
		Expected ProxyInstructions
	}{
		{
			Expected: ProxyInstructions{
				Failover: &Failover{
					Active: true,
					Codes:  []int{410, 411},
				},
				Upstreams: []*Upstream{
					&Upstream{
						Url: &url.URL{
							Scheme: "http",
							Host:   "localhost:5000",
							Path:   "/upstream",
						},
						Rates: []*Rate{},
					},
				},
			},
			Bytes: []byte(`{
    "failover": {"active": true, "codes": [410, 411]},
    "upstreams": [{
            "url": "http://localhost:5000/upstream"
        }]}`),
		},
		{
			Expected: ProxyInstructions{
				Failover: &Failover{
					Active: false,
					Codes:  nil,
				},
				Tokens: []*Token{
					&Token{
						Id: "Hello",
						Rates: []*Rate{
							&Rate{
								Increment: 1,
								Period:    time.Hour,
								Value:     10000,
							},
						},
					},
				},
				Upstreams: []*Upstream{
					&Upstream{
						Url: &url.URL{
							Scheme: "http",
							Host:   "localhost:5000",
							Path:   "/upstream",
						},
						Headers: map[string][]string{
							"X-Sasha":  []string{"b"},
							"X-Serega": []string{"a"},
						},
						Rates: []*Rate{
							&Rate{
								Increment: 1,
								Value:     10,
								Period:    time.Minute,
							},
						},
					},
					&Upstream{
						Url: &url.URL{
							Scheme: "http",
							Host:   "localhost:5000",
							Path:   "/upstream2",
						},
						Headers: map[string][]string{
							"X-Sasha":  []string{"b2"},
							"X-Serega": []string{"a2"},
						},
						Rates: []*Rate{
							&Rate{
								Increment: 1,
								Value:     4,
								Period:    time.Second,
							},
							&Rate{
								Increment: 1,
								Value:     40000,
								Period:    time.Minute,
							},
						},
					},
				},
			},
			Bytes: []byte(`{
    "tokens": [
        {
            "id": "Hello",
            "rates": [
                {
                    "increment": 1,
                    "period": "hour",
                    "value": 10000
                }
            ]
        }
    ],
    "upstreams": [
        {
            "headers": {
                "X-Sasha": [
                    "b"
                ],
                "X-Serega": [
                    "a"
                ]
            },
            "rates": [
                {
                    "increment": 1,
                    "period": "minute",
                    "value": 10
                }
            ],
            "url": "http://localhost:5000/upstream"
        },
        {
            "headers": {
                "X-Sasha": [
                    "b2"
                ],
                "X-Serega": [
                    "a2"
                ]
            },
            "rates": [
                {
                    "increment": 1,
                    "period": "second",
                    "value": 4
                },
                {
                    "increment": 1,
                    "period": "minute",
                    "value": 40000
                }
            ],
            "url": "http://localhost:5000/upstream2"
        }
    ]}`),
		},
	}
	for _, u := range objects {
		authResponse, err := ProxyInstructionsFromJson(u.Bytes)
		c.Assert(err, IsNil)
		//we will be checking individual elements here
		//as if something fails would be impossible to debug

		// Check failover
		c.Assert(authResponse.Failover, DeepEquals, u.Expected.Failover)

		//Check tokens
		c.Assert(len(authResponse.Tokens), Equals, len(u.Expected.Tokens))
		for i, token := range authResponse.Tokens {
			expectedToken := u.Expected.Tokens[i]
			c.Assert(token, DeepEquals, expectedToken)
		}

		//Check upstreams
		c.Assert(len(authResponse.Upstreams), Equals, len(u.Expected.Upstreams))
		for i, upstream := range authResponse.Upstreams {
			expectedUpstream := u.Expected.Upstreams[i]
			c.Assert(*upstream.Url, DeepEquals, *expectedUpstream.Url)
			c.Assert(upstream.Headers, DeepEquals, expectedUpstream.Headers)
			c.Assert(upstream.Rates, DeepEquals, expectedUpstream.Rates)
		}
	}
}

func (s *MarshalSuite) TestUnmarshalFail(c *C) {
	objects := [][]byte{
		//Empty
		[]byte(""),
		//Good json, bad format
		[]byte(`{}`),
		//bad upstream
		[]byte(`{"upstreams": [{}]}`),
		//bad rates in tokens
		[]byte(`{
    "tokens": [
        {
            "rates": [
                {
                    "increment": 1,
                    "period": "year", 
                    "value": -1
                }
            ], 
            "id": "hola"
        }
    ]
}`),
		//bad rates in upstreams
		[]byte(`{
    "upstreams": [
        {
            "rates": [
                {
                    "increment": 1,
                    "period": "super-minute",
                    "value": 10
                }
            ],
            "url": "http://localhost:5000/upstream"
        }
    ]
}`),
	}
	for _, bytes := range objects {
		_, err := ProxyInstructionsFromJson(bytes)
		c.Assert(err, NotNil)
	}
}
