package command

import (
	"encoding/json"
	. "launchpad.net/gocheck"
)

type FailoverSuite struct{}

var _ = Suite(&FailoverSuite{})

func (s *FailoverSuite) TestFailoverFromObj(c *C) {
	failovers := []struct {
		Expected Failover
		Parse    string
	}{
		{
			Parse: `true`,
			Expected: Failover{
				Active: true,
			},
		},
		{
			Parse: `false`,
			Expected: Failover{
				Active: false,
			},
		},
		{
			Parse: `{"active": true, "codes": [405, 503]}`,
			Expected: Failover{
				Active: true,
				Codes:  []int{405, 503},
			},
		},
	}

	for _, f := range failovers {
		var value interface{}
		err := json.Unmarshal([]byte(f.Parse), &value)
		parsed, err := NewFailoverFromObj(value)
		c.Assert(err, IsNil)
		c.Assert(*parsed, DeepEquals, f.Expected)
	}
}
