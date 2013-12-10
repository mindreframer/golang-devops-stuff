package backend

import (
	"github.com/mailgun/vulcan/timeutils"
	. "launchpad.net/gocheck"
	"testing"
	"time"
)

func Test2(t *testing.T) { TestingT(t) }

type MemoryBackendSuite struct {
	timeProvider *timeutils.FreezedTime
	backend      *MemoryBackend
}

var _ = Suite(&MemoryBackendSuite{})

func (s *MemoryBackendSuite) SetUpTest(c *C) {
	start := time.Date(2012, 3, 4, 5, 6, 7, 0, time.UTC)
	s.timeProvider = &timeutils.FreezedTime{CurrentTime: start}
	backend, err := NewMemoryBackend(s.timeProvider)
	c.Assert(err, IsNil)
	s.backend = backend
}

func (s *MemoryBackendSuite) TestUtcNow(c *C) {
	c.Assert(s.backend.UtcNow(), Equals, s.timeProvider.CurrentTime)
}

func (s *MemoryBackendSuite) TestMemoryBackendGetSet(c *C) {
	counter, err := s.backend.GetCount("key1", time.Second)
	c.Assert(err, IsNil)
	c.Assert(counter, Equals, int64(0))

	err = s.backend.UpdateCount("key1", time.Second, 2)
	c.Assert(err, IsNil)

	counter, err = s.backend.GetCount("key1", time.Second)
	c.Assert(err, IsNil)
	c.Assert(counter, Equals, int64(2))
}
