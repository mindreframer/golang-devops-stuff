package command

import (
	"encoding/json"
	. "launchpad.net/gocheck"
	"time"
)

type RateSuite struct{}

var _ = Suite(&RateSuite{})

func (s *RateSuite) TestNewRateSuccess(c *C) {
	rates := []struct {
		Units    int64
		Period   time.Duration
		UnitType int
		Expected *Rate
	}{
		{
			Units:    2,
			Period:   time.Second,
			UnitType: UnitTypeRequests,
			Expected: &Rate{Units: 2, Period: time.Second},
		},
		{
			Units:    10,
			UnitType: UnitTypeKilobytes,
			Period:   time.Minute,
			Expected: &Rate{Units: 10, Period: time.Minute, UnitType: UnitTypeKilobytes},
		},
	}

	for _, in := range rates {
		r, err := NewRate(in.Units, in.Period, in.UnitType)
		c.Assert(err, IsNil)
		c.Assert(*r, Equals, *in.Expected)
	}
}

func (s *RateSuite) TestNewRateFail(c *C) {
	rates := []struct {
		Units    int64
		Period   time.Duration
		UnitType int
	}{
		//period too small
		{
			Units:    1,
			UnitType: UnitTypeRequests,
			Period:   time.Millisecond,
		},
		//period too large
		{
			Units:    1,
			UnitType: UnitTypeRequests,
			Period:   time.Hour * 25,
		},
		//Zero not allowed
		{
			Units:    0,
			UnitType: UnitTypeRequests,
			Period:   time.Hour,
		},
		//Negative numbers
		{
			Units:    -1,
			UnitType: UnitTypeRequests,
			Period:   time.Hour,
		},
	}

	for _, u := range rates {
		_, err := NewRate(u.Units, u.Period, u.UnitType)
		c.Assert(err, NotNil)
	}
}

func (s *RateSuite) TestBuckets(c *C) {
	start := time.Date(2012, 3, 4, 5, 6, 7, 0, time.UTC)
	startMinutes := time.Date(2012, 3, 4, 5, 6, 0, 0, time.UTC)

	rates := []struct {
		Rate          Rate
		CurrentBucket time.Time
		NextBucket    time.Time
	}{
		{
			Rate: Rate{
				Units:  1,
				Period: time.Second,
			},
			CurrentBucket: start,
			NextBucket:    start.Add(time.Second),
		},
		{
			Rate: Rate{
				Units:  1,
				Period: time.Minute,
			},
			CurrentBucket: startMinutes,
			NextBucket:    startMinutes.Add(time.Minute),
		},
		{
			Rate: Rate{
				Units:  10,
				Period: time.Minute,
			},
			CurrentBucket: startMinutes,
			NextBucket:    startMinutes.Add(time.Minute),
		},
	}

	for _, u := range rates {
		c.Assert(u.Rate.CurrentBucket(start), Equals, u.CurrentBucket)
		c.Assert(u.Rate.NextBucket(start), Equals, u.NextBucket)
	}
}

func (s *RateSuite) TestRetrySeconds(c *C) {
	start := time.Date(2012, 3, 4, 5, 6, 7, 0, time.UTC)

	rates := []struct {
		Rate         Rate
		RetrySeconds int
	}{
		{
			Rate: Rate{
				Units:  1,
				Period: time.Second,
			},
			RetrySeconds: 1,
		},
		{
			Rate: Rate{
				Units:  1,
				Period: time.Minute,
			},
			RetrySeconds: 53,
		},
		{
			Rate: Rate{
				Units:  10,
				Period: time.Minute,
			},
			RetrySeconds: 53,
		},
		{
			Rate: Rate{
				Units:  1,
				Period: time.Hour,
			},
			RetrySeconds: 3233,
		},
	}

	for _, u := range rates {
		c.Assert(u.Rate.RetrySeconds(start), Equals, u.RetrySeconds)
	}
}

func (s *RateSuite) TestRateParsing(c *C) {
	rates := []struct {
		Rate  Rate
		Parse string
	}{
		{
			Rate: Rate{
				Units:    1,
				UnitType: UnitTypeRequests,
				Period:   time.Second,
			},
			Parse: `"1 request/second"`,
		},
		{
			Rate: Rate{
				Units:    2,
				UnitType: UnitTypeRequests,
				Period:   time.Minute,
			},
			Parse: `"2 requests/minute"`,
		},
		{
			Rate: Rate{
				Units:    3,
				UnitType: UnitTypeRequests,
				Period:   time.Hour,
			},
			Parse: `"3 reqs/hour"`,
		},
		{
			Rate: Rate{
				Units:    1,
				UnitType: UnitTypeRequests,
				Period:   time.Hour,
			},
			Parse: `"1 req/hour"`,
		},
		{
			Rate: Rate{
				Units:    1,
				UnitType: UnitTypeKilobytes,
				Period:   time.Hour,
			},
			Parse: `"1 KB/hour"`,
		},
		{
			Rate: Rate{
				Units:    12,
				UnitType: UnitTypeRequests,
				Period:   time.Second,
			},
			Parse: `{"requests": 12, "period": "second"}`,
		},
		{
			Rate: Rate{
				Units:    8,
				UnitType: UnitTypeRequests,
				Period:   time.Minute,
			},
			Parse: `{"requests": 8, "period": "minute"}`,
		},
		{
			Rate: Rate{
				Units:    10,
				UnitType: UnitTypeRequests,
				Period:   time.Hour,
			},
			Parse: `{"requests": 10, "period": "hour"}`,
		},
		{
			Rate: Rate{
				Units:    8,
				UnitType: UnitTypeKilobytes,
				Period:   time.Hour,
			},
			Parse: `{"KB": 8, "period": "hour"}`,
		},
	}
	for _, u := range rates {
		var value interface{}
		err := json.Unmarshal([]byte(u.Parse), &value)
		c.Assert(err, IsNil)
		parsed, err := NewRateFromObj(value)
		c.Assert(err, IsNil)
		c.Assert(u.Rate, Equals, *parsed)
	}
}

func (s *RateSuite) TestRateParsingFailures(c *C) {
	rates := []struct {
		Parse string
	}{
		// Invalid period
		{
			Parse: `"1 request/year"`,
		},
		// Invalid unit
		{
			Parse: `"1 parsec/year"`,
		},
		// Invalid format
		{
			Parse: `"1 /year"`,
		},
		// Invalid format
		{
			Parse: `"a b omag!afawf!@a3r53qt5q3rtqsge___awfg$W???-"`,
		},
		// Invalid data type
		{
			Parse: `["a", "b"]`,
		},
		// Invalid parameters
		{
			Parse: `{"apples": 2, "period": "second"}`,
		},
	}
	for _, u := range rates {
		var value interface{}
		err := json.Unmarshal([]byte(u.Parse), &value)
		c.Assert(err, IsNil)
		_, err = NewRateFromObj(value)
		c.Assert(err, Not(IsNil))
	}
}
