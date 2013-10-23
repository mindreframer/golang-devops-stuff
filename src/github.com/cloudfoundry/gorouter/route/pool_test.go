package route

import (
	. "launchpad.net/gocheck"
	"math"
)

type PSuite struct{}

func init() {
	Suite(&PSuite{})
}

func (s *PSuite) TestPoolAddingAndRemoving(c *C) {
	pool := NewPool()

	endpoint := &Endpoint{}

	pool.Add(endpoint)

	foundEndpoint, found := pool.Sample()
	c.Assert(found, Equals, true)
	c.Assert(foundEndpoint, Equals, endpoint)

	pool.Remove(endpoint)

	_, found = pool.Sample()
	c.Assert(found, Equals, false)
}

func (s *PSuite) TestPoolAddingDoesNotDuplicate(c *C) {
	pool := NewPool()

	endpoint := &Endpoint{}

	pool.Add(endpoint)
	pool.Add(endpoint)

	foundEndpoint, found := pool.Sample()
	c.Assert(found, Equals, true)
	c.Assert(foundEndpoint, Equals, endpoint)

	pool.Remove(endpoint)

	_, found = pool.Sample()
	c.Assert(found, Equals, false)
}

func (s *PSuite) TestPoolAddingEquivalentEndpointsDoesNotDuplicate(c *C) {
	pool := NewPool()

	endpoint1 := &Endpoint{Host: "1.2.3.4", Port: 5678}
	endpoint2 := &Endpoint{Host: "1.2.3.4", Port: 5678}

	pool.Add(endpoint1)
	pool.Add(endpoint2)

	_, found := pool.Sample()
	c.Assert(found, Equals, true)

	pool.Remove(endpoint1)

	_, found = pool.Sample()
	c.Assert(found, Equals, false)
}

func (s *PSuite) TestPoolIsEmptyInitially(c *C) {
	c.Assert(NewPool().IsEmpty(), Equals, true)
}

func (s *PSuite) TestPoolIsEmptyAfterRemovingEverything(c *C) {
	pool := NewPool()

	endpoint := &Endpoint{}

	pool.Add(endpoint)

	c.Assert(pool.IsEmpty(), Equals, false)

	pool.Remove(endpoint)

	c.Assert(pool.IsEmpty(), Equals, true)
}

func (s *PSuite) TestPoolFindByPrivateInstanceId(c *C) {
	pool := NewPool()

	endpointFoo := &Endpoint{Host: "1.2.3.4", Port: 1234, PrivateInstanceId: "foo"}
	endpointBar := &Endpoint{Host: "5.6.7.8", Port: 5678, PrivateInstanceId: "bar"}

	pool.Add(endpointFoo)
	pool.Add(endpointBar)

	foundEndpoint, found := pool.FindByPrivateInstanceId("foo")
	c.Assert(found, Equals, true)
	c.Assert(foundEndpoint, Equals, endpointFoo)

	foundEndpoint, found = pool.FindByPrivateInstanceId("bar")
	c.Assert(found, Equals, true)
	c.Assert(foundEndpoint, Equals, endpointBar)

	_, found = pool.FindByPrivateInstanceId("quux")
	c.Assert(found, Equals, false)
}

func (s *PSuite) TestPoolSamplingIsRandomish(c *C) {
	pool := NewPool()

	endpoint1 := &Endpoint{Host: "1.2.3.4", Port: 5678}
	endpoint2 := &Endpoint{Host: "5.6.7.8", Port: 1234}

	pool.Add(endpoint1)
	pool.Add(endpoint2)

	var occurrences1, occurrences2 int

	for i := 0; i < 200; i += 1 {
		foundEndpoint, _ := pool.Sample()
		if foundEndpoint == endpoint1 {
			occurrences1 += 1
		} else {
			occurrences2 += 1
		}
	}

	c.Assert(occurrences1, Not(Equals), 0)
	c.Assert(occurrences2, Not(Equals), 0)

	// they should be arbitrarily close
	c.Assert(math.Abs(float64(occurrences1-occurrences2)) < 50, Equals, true)
}

func (s *PSuite) TestPoolMarshalsAsJSON(c *C) {
	pool := NewPool()

	pool.Add(&Endpoint{Host: "1.2.3.4", Port: 5678})

	json, err := pool.MarshalJSON()
	c.Assert(err, IsNil)

	c.Assert(string(json), Equals, `["1.2.3.4:5678"]`)
}
