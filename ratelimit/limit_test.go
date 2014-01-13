package ratelimit

import (
	"github.com/mailgun/vulcan/backend"
	. "github.com/mailgun/vulcan/command"
	"github.com/mailgun/vulcan/timeutils"
	. "launchpad.net/gocheck"
	"testing"
	"time"
)

func TestRateLimit(t *testing.T) { TestingT(t) }

type LimitSuite struct {
	timeProvider       *timeutils.FreezedTime
	backend            *backend.MemoryBackend
	rateLimiter        RateLimiter
	failingRateLimiter RateLimiter
}

var _ = Suite(&LimitSuite{})

func (s *LimitSuite) SetUpTest(c *C) {
	start := time.Date(2012, 3, 4, 5, 6, 7, 0, time.UTC)
	s.timeProvider = &timeutils.FreezedTime{CurrentTime: start}
	b, err := backend.NewMemoryBackend(s.timeProvider)
	c.Assert(err, IsNil)
	s.backend = b
	s.rateLimiter = &BasicRateLimiter{Backend: s.backend}
	s.failingRateLimiter = &BasicRateLimiter{Backend: &backend.FailingBackend{}}
}

// Getting rates when there's no stats
func (s *LimitSuite) TestGetRatesNoStats(c *C) {
	rates := map[string][]*Rate{
		"a": []*Rate{
			&Rate{
				Units:  1,
				Period: time.Second * time.Duration(1),
			},
		},
	}

	seconds, err := s.rateLimiter.GetRetrySeconds(rates)
	c.Assert(err, IsNil)
	c.Assert(seconds, Equals, 0)
}

// Getting rates when there are some rates, but rate is not reached
func (s *LimitSuite) TestGetRatesStats(c *C) {
	s.backend.UpdateCount("a", time.Second*time.Duration(1), 1)

	rates := map[string][]*Rate{
		"a": []*Rate{
			&Rate{
				Units:  2,
				Period: time.Second * time.Duration(1),
			},
		},
	}

	seconds, err := s.rateLimiter.GetRetrySeconds(rates)
	c.Assert(err, IsNil)
	c.Assert(seconds, Equals, 0)
}

// Getting rates when there are some rates, rate limit is reached, retry is next second
func (s *LimitSuite) TestGetRatesLimitReached(c *C) {
	rates := map[string][]*Rate{
		"a": []*Rate{
			&Rate{
				Units:  1,
				Period: time.Second * time.Duration(1),
			},
		},
	}
	s.rateLimiter.UpdateStats(1, rates)

	seconds, err := s.rateLimiter.GetRetrySeconds(rates)
	c.Assert(err, IsNil)
	c.Assert(seconds, Equals, 1)
}

// Getting rates when there are some rates, rate limit is reached, retry is next minute
func (s *LimitSuite) TestGetRatesLimitReachedRetryNextMinute(c *C) {
	rates := map[string][]*Rate{
		"a": []*Rate{
			&Rate{
				Units:  1,
				Period: time.Minute * time.Duration(1),
			},
		},
	}

	s.rateLimiter.UpdateStats(1, rates)

	seconds, err := s.rateLimiter.GetRetrySeconds(rates)
	c.Assert(err, IsNil)
	c.Assert(seconds, Equals, 53)
}

// It takes only one rate to be reached for the request to be rate limited
func (s *LimitSuite) TestGetRatesLimitReachedTakesOne(c *C) {
	rates := map[string][]*Rate{
		"a": []*Rate{
			&Rate{
				Units:  1,
				Period: time.Minute * time.Duration(1),
			},
			&Rate{
				Units:  2,
				Period: time.Hour * time.Duration(1),
			},
		},
	}
	s.rateLimiter.UpdateStats(1, rates)
	seconds, err := s.rateLimiter.GetRetrySeconds(rates)
	c.Assert(err, IsNil)
	c.Assert(seconds, Equals, 53)
}

// It takes only one rate to be reached for the request to be rate limited
func (s *LimitSuite) TestGetRatesLimitReachedDifferentTokens(c *C) {
	rates := map[string][]*Rate{
		"a": []*Rate{
			&Rate{
				Units:  100,
				Period: time.Minute * time.Duration(1),
			},
		},
		"b": []*Rate{
			&Rate{
				Units:  1,
				Period: time.Hour * time.Duration(1),
			},
		},
	}
	s.rateLimiter.UpdateStats(1, rates)
	seconds, err := s.rateLimiter.GetRetrySeconds(rates)
	c.Assert(err, IsNil)
	c.Assert(seconds, Equals, 3233)
}

// Rate limited by max traffic capacity
func (s *LimitSuite) TestGetRatesLimitReachedTraffic(c *C) {
	rates := map[string][]*Rate{
		"a": []*Rate{
			&Rate{
				Units:    2,
				UnitType: UnitTypeKilobytes,
				Period:   time.Minute * time.Duration(1),
			},
			&Rate{
				Units:  10,
				Period: time.Hour * time.Duration(1),
			},
		},
	}
	s.rateLimiter.UpdateStats(4096, rates)
	seconds, err := s.rateLimiter.GetRetrySeconds(rates)
	c.Assert(err, IsNil)
	c.Assert(seconds, Equals, 53)
}
