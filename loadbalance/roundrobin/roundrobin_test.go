package roundrobin

import (
	"github.com/mailgun/vulcan/loadbalance"
	"github.com/mailgun/vulcan/timeutils"
	. "launchpad.net/gocheck"
	"testing"
	"time"
)

type E struct {
	id     string
	active bool
}

func (e *E) Id() string {
	return e.id
}

func (e *E) IsActive() bool {
	return e.active
}

func Test(t *testing.T) { TestingT(t) }

type RoundRobinSuite struct {
	timeProvider *timeutils.FreezedTime
}

var _ = Suite(&RoundRobinSuite{})

func (s *RoundRobinSuite) SetUpSuite(c *C) {
	currentDay := time.Date(2012, 3, 4, 5, 6, 7, 0, time.UTC)
	s.timeProvider = &timeutils.FreezedTime{CurrentTime: currentDay}
}

func (s *RoundRobinSuite) TestNoEndpoints(c *C) {
	r := NewRoundRobin(s.timeProvider)
	_, err := r.NextEndpoint([]loadbalance.Endpoint{})
	c.Assert(err, NotNil)
}

func (s *RoundRobinSuite) TestOneActiveEndpoint(c *C) {
	r := NewRoundRobin(s.timeProvider)
	epts := endpoints(&E{id: "a", active: true})

	e, err := r.NextEndpoint(epts)
	c.Assert(err, IsNil)
	endpoint := e.(*E)
	c.Assert(endpoint.id, Equals, "a")

	e, err = r.NextEndpoint(epts)
	endpoint = e.(*E)
	c.Assert(endpoint.id, Equals, "a")
}

func (s *RoundRobinSuite) TestOneUnavailableEndpoint(c *C) {
	r := NewRoundRobin(s.timeProvider)
	epts := endpoints(&E{id: "a", active: false})
	_, err := r.NextEndpoint(epts)
	c.Assert(err, NotNil)
}

func (s *RoundRobinSuite) TestRoundRobinPair(c *C) {
	r := NewRoundRobin(s.timeProvider)
	epts := endpoints(&E{id: "a", active: true}, &E{id: "b", active: true})

	e, err := r.NextEndpoint(epts)
	c.Assert(err, IsNil)
	endpoint := e.(*E)
	c.Assert(endpoint.id, Equals, "a")

	e, err = r.NextEndpoint(epts)
	endpoint = e.(*E)
	c.Assert(endpoint.id, Equals, "b")

	e, err = r.NextEndpoint(epts)
	endpoint = e.(*E)
	c.Assert(endpoint.id, Equals, "a")
}

func (s *RoundRobinSuite) TestRoundRobinEndpointGoesDown(c *C) {
	r := NewRoundRobin(s.timeProvider)
	a, b := &E{id: "a", active: true}, &E{id: "b", active: true}
	epts := endpoints(a, b)

	e, err := r.NextEndpoint(epts)
	c.Assert(err, IsNil)
	endpoint := e.(*E)
	c.Assert(endpoint.id, Equals, "a")

	e, err = r.NextEndpoint(epts)
	endpoint = e.(*E)
	c.Assert(endpoint.id, Equals, "b")

	// A goes down
	a.active = false
	e, err = r.NextEndpoint(epts)
	endpoint = e.(*E)
	c.Assert(endpoint.id, Equals, "b")

	// And it comes back
	a.active = true
	e, err = r.NextEndpoint(epts)
	endpoint = e.(*E)
	c.Assert(endpoint.id, Equals, "a")
}

func (s *RoundRobinSuite) TestRoundRobinMulti(c *C) {
	r := NewRoundRobin(s.timeProvider)
	a, b, z := &E{id: "a", active: true}, &E{id: "b", active: true}, &E{id: "c", active: true}
	epts := endpoints(a, b, z)

	e, _ := r.NextEndpoint(epts)
	c.Assert(e.Id(), Equals, "a")

	e, _ = r.NextEndpoint(epts)
	c.Assert(e.Id(), Equals, "b")

	e, _ = r.NextEndpoint(epts)
	c.Assert(e.Id(), Equals, "c")

	e, _ = r.NextEndpoint(epts)
	c.Assert(e.Id(), Equals, "a")
}

// Make sure that multiple endpoint combinations operate independently from each other
func (s *RoundRobinSuite) TestRoundRobinCombinations(c *C) {
	r := NewRoundRobin(s.timeProvider)
	a, b, z := &E{id: "a", active: true}, &E{id: "b", active: true}, &E{id: "c", active: true}
	epts := endpoints(a, b, z)
	epts2 := endpoints(a, z)

	e, _ := r.NextEndpoint(epts)
	c.Assert(e.Id(), Equals, "a")

	e, _ = r.NextEndpoint(epts2)
	c.Assert(e.Id(), Equals, "a")

	e, _ = r.NextEndpoint(epts)
	c.Assert(e.Id(), Equals, "b")

	e, _ = r.NextEndpoint(epts2)
	c.Assert(e.Id(), Equals, "c")

	e, _ = r.NextEndpoint(epts)
	c.Assert(e.Id(), Equals, "c")

	e, _ = r.NextEndpoint(epts2)
	c.Assert(e.Id(), Equals, "a")
}

// Make sure unused cursors are cleaned up
func (s *RoundRobinSuite) TestRoundRobinGc(c *C) {
	start := time.Date(2012, 3, 4, 5, 6, 7, 0, time.UTC)
	timeProvider := &timeutils.FreezedTime{CurrentTime: start}

	r := NewRoundRobin(timeProvider)
	a, b, z := &E{id: "a", active: true}, &E{id: "b", active: true}, &E{id: "c", active: true}
	epts := endpoints(a, b, z)
	epts2 := endpoints(a, z)

	e, _ := r.NextEndpoint(epts)
	c.Assert(e.Id(), Equals, "a")

	e, _ = r.NextEndpoint(epts2)
	c.Assert(e.Id(), Equals, "a")

	// in 10 seconds we've touched the second cursor, but not the first one
	timeProvider.CurrentTime = start.Add(time.Duration(10) * time.Second)

	e, _ = r.NextEndpoint(epts)
	c.Assert(e.Id(), Equals, "b")

	// Now it's time for gc, epts2 should go away
	timeProvider.CurrentTime = start.Add(time.Duration(ExpirySeconds+1) * time.Second)

	e, _ = r.NextEndpoint(epts)
	c.Assert(e.Id(), Equals, "c")

	// Let's inspect and make sure the first one was deleted
	c.Assert(len(r.cursors.cursors), Equals, 1)
	c.Assert(len(*r.cursors.expiryTimes), Equals, 1)

	// Make sure the remaining one is correct
	for _, cr := range r.cursors.cursors {
		c.Assert(cr[0].endpointIds, DeepEquals, []string{"a", "b", "c"})
	}
}

func endpoints(epts ...*E) []loadbalance.Endpoint {
	toReturn := make([]loadbalance.Endpoint, len(epts))
	for i, e := range epts {
		toReturn[i] = e
	}
	return toReturn
}
