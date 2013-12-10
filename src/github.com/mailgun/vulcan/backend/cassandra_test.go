package backend

import (
	"fmt"
	"github.com/mailgun/gocql"
	"github.com/mailgun/vulcan/timeutils"
	. "launchpad.net/gocheck"
	"os"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) }

type CassandraBackendSuite struct {
	timeProvider *timeutils.FreezedTime
	backend      *CassandraBackend
	shouldSkip   bool
	currentDay   time.Time
	previousDay  time.Time
}

var _ = Suite(&CassandraBackendSuite{})

func (b *CassandraBackendSuite) dropKeyspace(config *CassandraConfig) error {
	// first session creates a keyspace
	cluster := config.newCluster()
	cluster.Keyspace = ""
	session := cluster.CreateSession()
	defer session.Close()
	return session.Query(
		fmt.Sprintf(`DROP KEYSPACE %s`, config.Keyspace)).Exec()
}

func (s *CassandraBackendSuite) GetConfig() *CassandraConfig {
	cassandraConfig := &CassandraConfig{
		Servers:       []string{"localhost"},
		Keyspace:      "vulcan_test",
		Consistency:   gocql.One,
		LaunchCleanup: false,
	}
	return cassandraConfig
}

func (s *CassandraBackendSuite) SetUpTest(c *C) {
	if os.Getenv("CASSANDRA") != "yes" {
		s.shouldSkip = true
		return
	}
	s.currentDay = time.Date(2012, 3, 4, 5, 6, 7, 0, time.UTC)
	s.previousDay = time.Date(2012, 3, 3, 5, 6, 7, 0, time.UTC)

	s.timeProvider = &timeutils.FreezedTime{CurrentTime: s.currentDay}

	config := s.GetConfig()
	config.applyDefaults()
	s.dropKeyspace(config)

	backend, err := NewCassandraBackend(s.GetConfig(), s.timeProvider)
	c.Assert(err, IsNil)
	s.backend = backend
}

func (s *CassandraBackendSuite) TestUtcNow(c *C) {
	if s.shouldSkip {
		c.Skip("Cassandra backend is not activated")
	}
	c.Assert(s.backend.UtcNow(), Equals, s.timeProvider.CurrentTime)
}

// make sure the backend init is reentrable and does not alter existing data
func (s *CassandraBackendSuite) TestReentrable(c *C) {
	if s.shouldSkip {
		c.Skip("Cassandra backend is not activated")
	}

	_, err := NewCassandraBackend(s.GetConfig(), s.timeProvider)
	c.Assert(err, IsNil)

	_, err = NewCassandraBackend(s.GetConfig(), s.timeProvider)
	c.Assert(err, IsNil)
}

// Just make sure we can get and set stats
func (s *CassandraBackendSuite) TestBackendGetSet(c *C) {
	if s.shouldSkip {
		c.Skip("Cassandra backend is not activated")
	}

	counter, err := s.backend.GetCount("key1", time.Second)
	c.Assert(err, IsNil)
	c.Assert(counter, Equals, int64(0))

	err = s.backend.UpdateCount("key1", time.Second, 2)
	c.Assert(err, IsNil)

	counter, err = s.backend.GetCount("key1", time.Second)
	c.Assert(err, IsNil)
	c.Assert(counter, Equals, int64(2))
}

// Make sure cleanup procedure wipes out the data from last day and does not alter
// data from the previous day
func (s *CassandraBackendSuite) TestBackendCleanup(c *C) {

	if s.shouldSkip {
		c.Skip("Cassandra backend is not activated")
	}

	s.timeProvider.CurrentTime = s.previousDay
	err := s.backend.UpdateCount("key1", time.Second, 1)
	c.Assert(err, IsNil)

	s.timeProvider.CurrentTime = s.currentDay
	err = s.backend.UpdateCount("key1", time.Second, 2)
	c.Assert(err, IsNil)

	s.backend.cleanup()

	s.timeProvider.CurrentTime = s.currentDay
	counter, err := s.backend.GetCount("key1", time.Second)
	c.Assert(err, IsNil)
	c.Assert(counter, Equals, int64(2))

	s.timeProvider.CurrentTime = s.previousDay
	counter, err = s.backend.GetCount("key1", time.Second)
	c.Assert(err, IsNil)
	c.Assert(counter, Equals, int64(0))
}
