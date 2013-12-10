package vulcan

import (
	"fmt"
	"github.com/mailgun/vulcan/backend"
	. "github.com/mailgun/vulcan/instructions"
	"github.com/mailgun/vulcan/timeutils"
	. "launchpad.net/gocheck"
	"net/http"
	"time"
)

type ThrottlerSuite struct {
	timeProvider     *timeutils.FreezedTime
	backend          *backend.MemoryBackend
	throttler        *Throttler
	failingThrottler *Throttler
}

var _ = Suite(&ThrottlerSuite{})

func (s *ThrottlerSuite) SetUpTest(c *C) {
	start := time.Date(2012, 3, 4, 5, 6, 7, 0, time.UTC)
	s.timeProvider = &timeutils.FreezedTime{CurrentTime: start}
	b, err := backend.NewMemoryBackend(s.timeProvider)
	c.Assert(err, IsNil)
	s.backend = b
	s.throttler = NewThrottler(s.backend)

	s.failingThrottler = NewThrottler(&backend.FailingBackend{})
}

func (s *ThrottlerSuite) newUpstream(upstreamUrl string, rates ...*Rate) *Upstream {
	u, err := NewUpstream(upstreamUrl, rates, http.Header{})
	if err != nil {
		panic(err)
	}
	return u
}

func (s *ThrottlerSuite) newToken(tokenId string, rates ...*Rate) *Token {
	token, err := NewToken(tokenId, rates)
	if err != nil {
		panic(err)
	}
	return token
}

func getHit(now time.Time, key string, rate *Rate) string {
	return fmt.Sprintf(
		"%s_%s_%d", key, rate.Period.String(), rate.CurrentBucket(now).Unix())
}

func (s *ThrottlerSuite) rateBucket(key string, rate *Rate) string {
	return getHit(s.timeProvider.CurrentTime, key, rate)
}

func (s *ThrottlerSuite) upstreamId(stats []*UpstreamStats, index int) string {
	return stats[index].upstream.Id()
}

func (s *ThrottlerSuite) updateUpstreamStats(upstream *Upstream) {
	err := s.throttler.updateUpstreamStats(upstream)
	if err != nil {
		panic(err)
	}
}

func (s *ThrottlerSuite) updateTokenStats(token *Token) {
	err := s.throttler.updateTokenStats(token)
	if err != nil {
		panic(err)
	}
}

// Make sure handle the case when there's nothing to update
func (s *ThrottlerSuite) TestThrottlerUpdateStatsNoRates(c *C) {
	upstream := s.newUpstream("http://yahoo.com")
	tokens := []*Token{}
	s.throttler.updateStats(tokens, upstream)
}

// Make sure stats are properly updated
func (s *ThrottlerSuite) TestThrottlerUpdateStatsRates(c *C) {
	up := s.newUpstream("http://yahoo.com", &Rate{Increment: 1, Value: 1, Period: time.Second}, &Rate{Increment: 1, Value: 1, Period: time.Minute})
	tokens := []*Token{s.newToken("a", &Rate{Increment: 1, Value: 1, Period: time.Minute}, &Rate{Increment: 1, Value: 10, Period: time.Hour})}
	s.throttler.updateStats(tokens, up)
	token := tokens[0]

	expected := map[string]int64{
		s.rateBucket(up.Id(), up.Rates[0]):     1,
		s.rateBucket(up.Id(), up.Rates[1]):     1,
		s.rateBucket(token.Id, token.Rates[0]): 1,
		s.rateBucket(token.Id, token.Rates[1]): 1,
	}

	c.Assert(len(s.backend.Hits), Equals, len(expected))
	for key, val := range expected {
		c.Assert(s.backend.Hits[key], Equals, val)
	}
}

// Make sure stats are properly updated
func (s *ThrottlerSuite) TestThrottlerUpdateStatsFails(c *C) {
	up := s.newUpstream("http://yahoo.com", &Rate{Increment: 1, Value: 1, Period: time.Second}, &Rate{Value: 1, Period: time.Minute})
	tokens := []*Token{s.newToken("a", &Rate{Increment: 1, Value: 1, Period: time.Minute}, &Rate{Value: 10, Period: time.Hour})}
	err := s.failingThrottler.updateStats(tokens, up)
	c.Assert(err, NotNil)
}

// Make sure upstreams without rates are ok
func (s *ThrottlerSuite) TestThrottlerUpstreamsNoRates(c *C) {
	instructions := &ProxyInstructions{
		Upstreams: []*Upstream{
			s.newUpstream("http://google.com"),
			s.newUpstream("http://yahoo.com"),
		},
	}

	upstreamStats, _, err := s.throttler.throttle(instructions)

	c.Assert(err, IsNil)
	c.Assert(2, Equals, len(upstreamStats))
	c.Assert(s.upstreamId(upstreamStats, 0), Equals, "http://google.com")
	c.Assert(s.upstreamId(upstreamStats, 1), Equals, "http://yahoo.com")
}

// Rates present, but not hit, as there's no usage
func (s *ThrottlerSuite) TestThrottlerRatesClear(c *C) {
	instructions := &ProxyInstructions{
		Upstreams: []*Upstream{
			s.newUpstream("http://google.com", &Rate{Increment: 1, Value: 1, Period: time.Second}),
			s.newUpstream("http://yahoo.com", &Rate{Increment: 1, Value: 10, Period: time.Minute}),
		},
	}

	upstreamStats, _, err := s.throttler.throttle(instructions)

	c.Assert(err, IsNil)
	c.Assert(2, Equals, len(upstreamStats))

	c.Assert(s.upstreamId(upstreamStats, 0), Equals, "http://google.com")
	c.Assert(s.upstreamId(upstreamStats, 1), Equals, "http://yahoo.com")
}

// One upstream is out as the rate usage has been exceeded
func (s *ThrottlerSuite) TestThrottlerRatesOneUpstreamOut(c *C) {
	instructions := &ProxyInstructions{
		Upstreams: []*Upstream{
			s.newUpstream("http://google.com", &Rate{Increment: 1, Value: 1, Period: time.Second}),
			s.newUpstream("http://yahoo.com", &Rate{Increment: 1, Value: 10, Period: time.Minute}),
		},
	}

	s.updateUpstreamStats(instructions.Upstreams[0])

	upstreamStats, _, err := s.throttler.throttle(instructions)

	c.Assert(err, IsNil)
	c.Assert(len(upstreamStats), Equals, 2)
	c.Assert(upstreamStats[0].ExceededLimits(), Equals, true)
	c.Assert(upstreamStats[1].ExceededLimits(), Equals, false)
}

// All upstreams are out, make sure that retryTime is calculated properly
func (s *ThrottlerSuite) TestThrottlerRatesAllUpstreamsOut(c *C) {
	instructions := &ProxyInstructions{
		Upstreams: []*Upstream{
			s.newUpstream("http://google.com", &Rate{Increment: 1, Value: 1, Period: time.Second}),
			s.newUpstream("http://yahoo.com", &Rate{Increment: 10, Value: 10, Period: time.Minute}),
		},
	}

	s.updateUpstreamStats(instructions.Upstreams[0])
	s.updateUpstreamStats(instructions.Upstreams[1])

	upstreamStats, retrySeconds, err := s.throttler.throttle(instructions)

	c.Assert(err, IsNil)
	c.Assert(len(upstreamStats), Equals, 2)
	c.Assert(retrySeconds, Equals, 1)
}

// All upstreams are out, make sure retry time would be:
// * The time of the earliest available upstream
// * As upstreams have multiple rates, the rate should be calculated correctly
// taking the next available time of the slowest rate
func (s *ThrottlerSuite) TestThrottlerRatesAllUpstreamsOutSlowestRate(c *C) {
	instructions := &ProxyInstructions{
		Upstreams: []*Upstream{
			s.newUpstream("http://google.com", &Rate{Increment: 1, Value: 1, Period: time.Second}, &Rate{Increment: 1, Value: 1, Period: time.Hour}),
			s.newUpstream("http://yahoo.com", &Rate{Increment: 12, Value: 10, Period: time.Minute}, &Rate{Increment: 1, Value: 1, Period: time.Duration(10) * time.Hour}),
		},
	}

	s.updateUpstreamStats(instructions.Upstreams[0])
	s.updateUpstreamStats(instructions.Upstreams[1])

	upstreamStats, retrySeconds, err := s.throttler.throttle(instructions)

	c.Assert(err, IsNil)
	c.Assert(len(upstreamStats), Equals, 2)
	// seconds till next hour since s.timeProvider.CurrentTime
	c.Assert(retrySeconds, Equals, 3233)
}

// Backend is down, make sure there's no panic and we returned error code
func (s *ThrottlerSuite) TestThrottlerBackendFails(c *C) {
	instructions := &ProxyInstructions{
		Upstreams: []*Upstream{
			s.newUpstream("http://google.com", &Rate{Increment: 1, Value: 1, Period: time.Second}),
			s.newUpstream("http://yahoo.com", &Rate{Increment: 10, Value: 10, Period: time.Minute}),
		},
	}

	upstreamStats, _, err := s.failingThrottler.throttle(instructions)
	c.Assert(err, NotNil)
	c.Assert(len(upstreamStats), Equals, 0)
}

// Make sure upstreams and tokens without rates are ok
func (s *ThrottlerSuite) TestTokensAndUpstreams(c *C) {
	instructions := &ProxyInstructions{
		Tokens: []*Token{
			s.newToken("a"),
			s.newToken("b"),
		},
		Upstreams: []*Upstream{
			s.newUpstream("http://google.com"),
			s.newUpstream("http://yahoo.com"),
		},
	}

	upstreamStats, _, err := s.throttler.throttle(instructions)

	c.Assert(err, IsNil)
	c.Assert(2, Equals, len(upstreamStats))
	c.Assert(s.upstreamId(upstreamStats, 0), Equals, "http://google.com")
	c.Assert(s.upstreamId(upstreamStats, 1), Equals, "http://yahoo.com")
}

// Make sure upstreams and tokens with rates, but no usage
func (s *ThrottlerSuite) TestTokensAndUpstreamsWithRates(c *C) {
	instructions := &ProxyInstructions{
		Tokens: []*Token{
			s.newToken("a", &Rate{Value: 1, Period: time.Second}),
			s.newToken("b", &Rate{Value: 10, Period: time.Second}),
		},
		Upstreams: []*Upstream{
			s.newUpstream("http://google.com", &Rate{Value: 1, Period: time.Second}),
			s.newUpstream("http://yahoo.com", &Rate{Increment: 1, Value: 10, Period: time.Minute}),
		},
	}

	upstreamStats, _, err := s.throttler.throttle(instructions)

	c.Assert(err, IsNil)
	c.Assert(2, Equals, len(upstreamStats))
	c.Assert(s.upstreamId(upstreamStats, 0), Equals, "http://google.com")
	c.Assert(s.upstreamId(upstreamStats, 1), Equals, "http://yahoo.com")
}

// One token is out, means that all upstreams are out
func (s *ThrottlerSuite) TestTokensTokenIsOut(c *C) {
	instructions := &ProxyInstructions{
		Tokens: []*Token{
			s.newToken("a", &Rate{Increment: 1, Value: 1, Period: time.Second}),
			s.newToken("b", &Rate{Increment: 1, Value: 10, Period: time.Second}),
		},
		Upstreams: []*Upstream{
			s.newUpstream("http://google.com"),
			s.newUpstream("http://yahoo.com"),
		},
	}

	s.updateTokenStats(instructions.Tokens[0])

	upstreamStats, retrySeconds, err := s.throttler.throttle(instructions)

	c.Assert(err, IsNil)
	c.Assert(len(upstreamStats), Equals, 0)
	c.Assert(retrySeconds, Equals, 1)
}

// Both tokens are out, note that retry seconds is defined by the
// slowest rate of the slowest token
func (s *ThrottlerSuite) TestTokensAllTokenAreOut(c *C) {
	instructions := &ProxyInstructions{
		Tokens: []*Token{
			s.newToken("a", &Rate{Increment: 2, Value: 1, Period: time.Second}, &Rate{Increment: 2, Value: 2, Period: time.Minute}),
			s.newToken("b", &Rate{Increment: 1, Value: 10, Period: time.Second}, &Rate{Increment: 1, Value: 1, Period: time.Hour}),
		},
		Upstreams: []*Upstream{
			s.newUpstream("http://google.com"),
			s.newUpstream("http://yahoo.com"),
		},
	}

	s.updateTokenStats(instructions.Tokens[0])
	s.updateTokenStats(instructions.Tokens[1])

	upstreamStats, retrySeconds, err := s.throttler.throttle(instructions)

	c.Assert(err, IsNil)
	c.Assert(len(upstreamStats), Equals, 0)
	c.Assert(retrySeconds, Equals, 3233)
}

func (s *ThrottlerSuite) TestTokensFailingBackend(c *C) {
	instructions := &ProxyInstructions{
		Tokens: []*Token{
			s.newToken("a", &Rate{Increment: 1, Value: 1, Period: time.Second}),
			s.newToken("b", &Rate{Increment: 1, Value: 10, Period: time.Second}),
		},
		Upstreams: []*Upstream{
			s.newUpstream("http://google.com"),
			s.newUpstream("http://yahoo.com"),
		},
	}

	_, _, err := s.failingThrottler.throttle(instructions)
	c.Assert(err, NotNil)
}
