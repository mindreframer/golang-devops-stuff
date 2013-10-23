package registry

import (
	"encoding/json"
	"time"

	"github.com/cloudfoundry/go_cfmessagebus/mock_cfmessagebus"
	. "launchpad.net/gocheck"

	"github.com/cloudfoundry/gorouter/config"
	"github.com/cloudfoundry/gorouter/route"
)

type RegistrySuite struct {
	*Registry

	messageBus *mock_cfmessagebus.MockMessageBus
}

var _ = Suite(&RegistrySuite{})

var fooEndpoint, barEndpoint, bar2Endpoint *route.Endpoint

func (s *RegistrySuite) SetUpTest(c *C) {
	var configObj *config.Config

	configObj = config.DefaultConfig()
	configObj.DropletStaleThreshold = 10 * time.Millisecond

	s.messageBus = mock_cfmessagebus.NewMockMessageBus()
	s.Registry = NewRegistry(configObj, s.messageBus)

	fooEndpoint = &route.Endpoint{
		Host: "192.168.1.1",
		Port: 1234,

		ApplicationId: "12345",
		Tags: map[string]string{
			"runtime":   "ruby18",
			"framework": "sinatra",
		},
	}

	barEndpoint = &route.Endpoint{
		Host: "192.168.1.2",
		Port: 4321,

		ApplicationId: "54321",
		Tags: map[string]string{
			"runtime":   "javascript",
			"framework": "node",
		},
	}

	bar2Endpoint = &route.Endpoint{
		Host: "192.168.1.3",
		Port: 1234,

		ApplicationId: "54321",
		Tags: map[string]string{
			"runtime":   "javascript",
			"framework": "node",
		},
	}
}

func (s *RegistrySuite) TestRegister(c *C) {
	s.Register("foo", fooEndpoint)
	s.Register("fooo", fooEndpoint)
	c.Check(s.NumUris(), Equals, 2)
	firstUpdateTime := s.timeOfLastUpdate

	s.Register("bar", barEndpoint)
	s.Register("baar", barEndpoint)
	c.Check(s.NumUris(), Equals, 4)
	secondUpdateTime := s.timeOfLastUpdate

	c.Assert(secondUpdateTime.After(firstUpdateTime), Equals, true)
}

func (s *RegistrySuite) TestRegisterIgnoreDuplicates(c *C) {
	s.Register("bar", barEndpoint)
	s.Register("baar", barEndpoint)

	c.Check(s.NumUris(), Equals, 2)
	c.Check(s.NumEndpoints(), Equals, 1)

	s.Register("bar", barEndpoint)
	s.Register("baar", barEndpoint)

	c.Check(s.NumUris(), Equals, 2)
	c.Check(s.NumEndpoints(), Equals, 1)

	s.Unregister("bar", barEndpoint)

	c.Check(s.NumUris(), Equals, 1)
	c.Check(s.NumEndpoints(), Equals, 1)

	s.Unregister("baar", barEndpoint)

	c.Check(s.NumUris(), Equals, 0)
	c.Check(s.NumEndpoints(), Equals, 0)
}

func (s *RegistrySuite) TestRegisterUppercase(c *C) {
	m1 := &route.Endpoint{
		Host: "192.168.1.1",
		Port: 1234,
	}

	m2 := &route.Endpoint{
		Host: "192.168.1.1",
		Port: 1235,
	}

	s.Register("foo", m1)
	s.Register("FOO", m2)

	c.Check(s.NumUris(), Equals, 1)
}

func (s *RegistrySuite) TestRegisterDoesntReplaceSameEndpoint(c *C) {
	m1 := &route.Endpoint{
		Host: "192.168.1.1",
		Port: 1234,
	}

	m2 := &route.Endpoint{
		Host: "192.168.1.1",
		Port: 1234,
	}

	s.Register("foo", m1)
	s.Register("bar", m2)

	c.Check(s.NumUris(), Equals, 2)
	c.Check(s.NumEndpoints(), Equals, 1)
}

func (s *RegistrySuite) TestUnregister(c *C) {
	s.Register("bar", barEndpoint)
	s.Register("baar", barEndpoint)
	c.Check(s.NumUris(), Equals, 2)
	c.Check(s.NumEndpoints(), Equals, 1)

	s.Register("bar", bar2Endpoint)
	s.Register("baar", bar2Endpoint)
	c.Check(s.NumUris(), Equals, 2)
	c.Check(s.NumEndpoints(), Equals, 2)

	s.Unregister("bar", barEndpoint)
	s.Unregister("baar", barEndpoint)
	c.Check(s.NumUris(), Equals, 2)
	c.Check(s.NumEndpoints(), Equals, 1)

	s.Unregister("bar", bar2Endpoint)
	s.Unregister("baar", bar2Endpoint)
	c.Check(s.NumUris(), Equals, 0)
	c.Check(s.NumEndpoints(), Equals, 0)
}

func (s *RegistrySuite) TestUnregisterUppercase(c *C) {
	m1 := &route.Endpoint{
		Host: "192.168.1.1",
		Port: 1234,
	}

	m2 := &route.Endpoint{
		Host: "192.168.1.1",
		Port: 1234,
	}

	s.Register("foo", m1)
	s.Unregister("FOO", m2)

	c.Check(s.NumUris(), Equals, 0)
}

func (s *RegistrySuite) TestUnregisterDoesntDemolish(c *C) {
	m1 := &route.Endpoint{
		Host: "192.168.1.1",
		Port: 1234,
	}

	m2 := &route.Endpoint{
		Host: "192.168.1.1",
		Port: 1234,
	}

	s.Register("foo", m1)
	s.Register("bar", m1)

	s.Unregister("foo", m2)

	c.Check(s.NumUris(), Equals, 1)
}

func (s *RegistrySuite) TestLookup(c *C) {
	m := &route.Endpoint{
		Host: "192.168.1.1",
		Port: 1234,
	}

	s.Register("foo", m)

	var b *route.Endpoint
	var ok bool

	b, ok = s.Lookup("foo")
	c.Assert(ok, Equals, true)
	c.Check(b.CanonicalAddr(), Equals, "192.168.1.1:1234")

	b, ok = s.Lookup("FOO")
	c.Assert(ok, Equals, true)
	c.Check(b.CanonicalAddr(), Equals, "192.168.1.1:1234")
}

func (s *RegistrySuite) TestLookupDoubleRegister(c *C) {
	m1 := &route.Endpoint{
		Host: "192.168.1.2",
		Port: 1234,
	}

	m2 := &route.Endpoint{
		Host: "192.168.1.2",
		Port: 1235,
	}

	s.Register("bar", m1)
	s.Register("barr", m1)

	s.Register("bar", m2)
	s.Register("barr", m2)

	c.Check(s.NumUris(), Equals, 2)
	c.Check(s.NumEndpoints(), Equals, 2)
}

func (s *RegistrySuite) TestPruneStaleApps(c *C) {
	s.Register("foo", fooEndpoint)
	s.Register("fooo", fooEndpoint)

	s.Register("bar", barEndpoint)
	s.Register("baar", barEndpoint)

	c.Check(s.NumUris(), Equals, 4)
	c.Check(s.NumEndpoints(), Equals, 2)

	time.Sleep(s.dropletStaleThreshold + 1*time.Millisecond)
	s.PruneStaleDroplets()

	s.Register("bar", bar2Endpoint)
	s.Register("baar", bar2Endpoint)

	c.Check(s.NumUris(), Equals, 2)
	c.Check(s.NumEndpoints(), Equals, 1)
}

func (s *RegistrySuite) TestPruningIsByUriNotJustAddr(c *C) {
	endpoint := &route.Endpoint{
		Host: "192.168.1.1",
		Port: 1234,
	}

	s.Register("foo", endpoint)
	s.Register("bar", endpoint)

	s.Register("foo", endpoint)

	c.Check(s.NumUris(), Equals, 2)
	c.Check(s.NumEndpoints(), Equals, 1)

	time.Sleep(s.dropletStaleThreshold + 1*time.Millisecond)

	s.Register("foo", endpoint)

	s.PruneStaleDroplets()

	c.Check(s.NumUris(), Equals, 1)
	c.Check(s.NumEndpoints(), Equals, 1)

	foundEndpoint, found := s.Lookup("foo")
	c.Check(found, Equals, true)
	c.Check(foundEndpoint, DeepEquals, endpoint)

	_, found = s.Lookup("bar")
	c.Check(found, Equals, false)
}

func (s *RegistrySuite) TestPruneStaleAppsWhenStateStale(c *C) {
	s.Register("foo", fooEndpoint)
	s.Register("fooo", fooEndpoint)

	s.Register("bar", barEndpoint)
	s.Register("baar", barEndpoint)

	c.Check(s.NumUris(), Equals, 4)
	c.Check(s.NumEndpoints(), Equals, 2)

	time.Sleep(s.dropletStaleThreshold + 1*time.Millisecond)

	s.messageBus.OnPing(func() bool { return false })

	time.Sleep(s.dropletStaleThreshold + 1*time.Millisecond)

	s.PruneStaleDroplets()

	c.Check(s.NumUris(), Equals, 4)
	c.Check(s.NumEndpoints(), Equals, 2)
}

func (s *RegistrySuite) TestPruneStaleDropletsDoesNotDeadlock(c *C) {
	// when pruning stale droplets,
	// and the stale check takes a while,
	// and a read request comes in (i.e. from Lookup),
	// the read request completes before the stale check

	s.Register("foo", fooEndpoint)
	s.Register("fooo", fooEndpoint)

	completeSequence := make(chan string)

	s.messageBus.OnPing(func() bool {
		time.Sleep(5 * time.Second)
		completeSequence <- "stale"
		return false
	})

	go s.PruneStaleDroplets()

	go func() {
		s.Lookup("foo")
		completeSequence <- "lookup"
	}()

	firstCompleted := <-completeSequence

	c.Assert(firstCompleted, Equals, "lookup")
}

func (s *RegistrySuite) TestInfoMarshalling(c *C) {
	m := &route.Endpoint{
		Host: "192.168.1.1",
		Port: 1234,
	}

	s.Register("foo", m)
	marshalled, err := json.Marshal(s)
	c.Check(err, IsNil)

	c.Check(string(marshalled), Equals, "{\"foo\":[\"192.168.1.1:1234\"]}")
}
