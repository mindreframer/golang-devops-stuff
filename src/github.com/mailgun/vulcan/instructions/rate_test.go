package instructions

import (
	. "launchpad.net/gocheck"
	"testing"
	"time"
)

func TestRates(t *testing.T) { TestingT(t) }

type RateSuite struct{}

var _ = Suite(&RateSuite{})

func (s *RateSuite) TestNewRateSuccess(c *C) {
	rates := []struct {
		Value     int64
		Increment int64
		Period    time.Duration
		Expected  Rate
	}{
		{
			Value:     1,
			Increment: 2,
			Period:    time.Second,
			Expected:  Rate{Increment: 2, Value: 1, Period: time.Second},
		},
		{
			Value:     10,
			Increment: 3,
			Period:    time.Minute,
			Expected:  Rate{Increment: 3, Value: 10, Period: time.Minute},
		},
	}

	for _, u := range rates {
		r, err := NewRate(u.Increment, u.Value, u.Period)
		c.Assert(err, IsNil)
		c.Assert(*r, Equals, u.Expected)
	}
}

func (s *RateSuite) TestNewRateFail(c *C) {
	rates := []struct {
		Value     int64
		Increment int64
		Period    time.Duration
	}{
		//period too small
		{
			Value:     1,
			Increment: 10,
			Period:    time.Millisecond,
		},
		//period too large
		{
			Value:     1,
			Increment: 1,
			Period:    time.Hour * 25,
		},
		//Zero not allowed
		{
			Value:     0,
			Increment: 1,
			Period:    time.Hour,
		},
		//Zero not allowed
		{
			Value:     1,
			Increment: 0,
			Period:    time.Hour,
		},
		//Negative numbers
		{
			Value:     -1,
			Increment: 1,
			Period:    time.Hour,
		},
	}

	for _, u := range rates {
		_, err := NewRate(u.Increment, u.Value, u.Period)
		c.Assert(err, NotNil)
	}
}

func (s *RateSuite) TestPeriodSecondsAndDuration(c *C) {
	rates := []struct {
		Rate     Rate
		Seconds  int64
		Duration time.Duration
	}{
		{
			Rate: Rate{
				Value:     1,
				Increment: 1,
				Period:    time.Second,
			},
			Seconds:  1,
			Duration: time.Duration(time.Second) * time.Duration(1),
		},
		{
			Rate: Rate{
				Increment: 1,
				Value:     2,
				Period:    time.Hour,
			},
			Seconds:  7200,
			Duration: time.Duration(time.Hour) * time.Duration(2),
		},
		{
			Rate: Rate{
				Value:     10,
				Increment: 1,
				Period:    time.Minute,
			},
			Seconds:  600,
			Duration: time.Duration(time.Minute) * time.Duration(10),
		},
	}

	for _, u := range rates {
		c.Assert(u.Rate.PeriodSeconds(), Equals, u.Seconds)
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
				Value:     1,
				Increment: 1,
				Period:    time.Second,
			},
			CurrentBucket: start,
			NextBucket:    start.Add(time.Second),
		},
		{
			Rate: Rate{
				Value:     1,
				Increment: 1,
				Period:    time.Minute,
			},
			CurrentBucket: startMinutes,
			NextBucket:    startMinutes.Add(time.Minute),
		},
		{
			Rate: Rate{
				Value:     10,
				Increment: 1,
				Period:    time.Minute,
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
				Value:     1,
				Increment: 1,
				Period:    time.Second,
			},
			RetrySeconds: 1,
		},
		{
			Rate: Rate{
				Value:     1,
				Increment: 1,
				Period:    time.Minute,
			},
			RetrySeconds: 53,
		},
		{
			Rate: Rate{
				Value:     10,
				Increment: 1,
				Period:    time.Minute,
			},
			RetrySeconds: 53,
		},
		{
			Rate: Rate{
				Value:     1,
				Increment: 1,
				Period:    time.Hour,
			},
			RetrySeconds: 3233,
		},
	}

	for _, u := range rates {
		c.Assert(u.Rate.RetrySeconds(start), Equals, u.RetrySeconds)
	}
}
