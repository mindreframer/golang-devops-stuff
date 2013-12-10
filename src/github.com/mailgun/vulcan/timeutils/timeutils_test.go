package timeutils

import (
	"fmt"
	. "launchpad.net/gocheck"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) }

type TimeUtilsSuite struct {
	timeProvider TimeProvider
}

var _ = Suite(&TimeUtilsSuite{})

func (s *TimeUtilsSuite) SetUpSuite(c *C) {
	start := time.Date(2012, 3, 4, 5, 6, 7, 0, time.UTC)
	s.timeProvider = &FreezedTime{CurrentTime: start}
}

func (s *TimeUtilsSuite) TestEpochDay(c *C) {
	dates := []struct {
		Date time.Time
		Day  int64
	}{
		{
			Date: time.Date(2012, 3, 4, 5, 6, 7, 0, time.UTC),
			Day:  15403,
		},
		{
			Date: time.Date(2012, 3, 4, 5, 6, 7, 12, time.UTC),
			Day:  15403,
		},
		{
			Date: time.Date(2012, 3, 4, 5, 6, 59, 12, time.UTC),
			Day:  15403,
		},
		{
			Date: time.Date(2012, 3, 4, 5, 59, 59, 12, time.UTC),
			Day:  15403,
		},
		{
			Date: time.Date(2012, 3, 4, 9, 59, 59, 12, time.UTC),
			Day:  15403,
		},
		{
			Date: time.Date(2012, 3, 5, 9, 59, 59, 12, time.UTC),
			Day:  15404,
		},
	}
	for _, t := range dates {
		c.Assert(EpochDay(t.Date), Equals, t.Day)
	}
}

func (s *TimeUtilsSuite) TestGetHit(c *C) {
	hits := []struct {
		Key      string
		Period   time.Duration
		Expected string
	}{
		{
			Key:      "key1",
			Period:   time.Second,
			Expected: "key1_1s_%d",
		},
		{
			Key:      "key2",
			Period:   time.Minute,
			Expected: "key2_1m0s_%d",
		},
		{
			Key:      "key1",
			Period:   time.Hour,
			Expected: "key1_1h0m0s_%d",
		},
	}
	for _, u := range hits {
		expected := fmt.Sprintf(u.Expected, RoundedBucket(s.timeProvider.UtcNow(), u.Period).Unix())
		hit := GetHit(s.timeProvider.UtcNow(), u.Key, u.Period)
		c.Assert(expected, Equals, hit)
	}
}

func (s *TimeUtilsSuite) TestTimes(c *C) {
	tm := &RealTime{}
	c.Assert(tm.UtcNow(), NotNil)
}
