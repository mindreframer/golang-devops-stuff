/*
Cassandra backend based on counters. As long as counters in cassandra can not be TTLed,
this backend performs periodic garbage collections.

The implementation is pretty straightforward:

* Every counter in every time period is represented as a separate row
* There are two tables: hits_even and hits_odd
* Every day is numbered as even or odd (day number since epoch)
* Even day counters are stored in hits_even, odd in hits_odd
* The table that is not currently updated gets truncated once a day

*/
package backend

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/mailgun/gocql"
	"github.com/mailgun/vulcan/timeutils"
	"strings"
	"time"
)

type CassandraConfig struct {
	Keyspace          string
	Consistency       gocql.Consistency
	Servers           []string
	ConnectTimeout    time.Duration
	ReplicationFactor int
	LaunchCleanup     bool
	CleanupTime       *CleanupTime
}

type CleanupTime struct {
	Hour   int
	Minute int
}

type CassandraBackend struct {
	session      *gocql.Session // session
	timeProvider timeutils.TimeProvider
	config       *CassandraConfig
}

func (b *CassandraBackend) GetCount(key string, period time.Duration) (int64, error) {
	var counter int64
	hitKey := timeutils.GetHit(b.UtcNow(), key, period)
	activeTable, _ := b.tableNames()
	query := b.session.Query(
		fmt.Sprintf(
			"SELECT value from %s WHERE hit = ? LIMIT 1",
			activeTable),
		hitKey)

	if err := query.Scan(&counter); err != nil {
		if err == gocql.ErrNotFound {
			glog.Infof("Entry %s for %s not found, it's ok", key, hitKey)
			return 0, nil
		}
		glog.Errorf("Error when executing query, err: %s", err)
		return -1, err
	}
	glog.Infof("Hitkey: %s counter: %d", hitKey, counter)
	return counter, nil
}

func (b *CassandraBackend) UpdateCount(key string, period time.Duration, increment int64) error {
	activeTable, _ := b.tableNames()
	hitKey := timeutils.GetHit(b.UtcNow(), key, period)
	query := b.session.Query(
		fmt.Sprintf(
			"UPDATE %s SET value = value + ? WHERE hit = ?",
			activeTable),
		increment,
		hitKey)

	if err := query.Exec(); err != nil {
		glog.Errorf("Error when executing update query for %s, err: %s", hitKey, err)
		return err
	}
	return nil
}

// Standard dial and read timeouts, can be overriden when supplying
// proxy settings
const (
	DefaultConnectTimeout    = time.Duration(200) * time.Millisecond
	DefaultReplicationFactor = 3
	DefaultCleanupHour       = 0
	DefaultCleanupMinute     = 30
)

func NewCassandraBackend(config *CassandraConfig, timeProvider timeutils.TimeProvider) (*CassandraBackend, error) {
	err := config.applyDefaults()
	if err != nil {
		return nil, err
	}

	backend := &CassandraBackend{
		timeProvider: timeProvider,
		config:       config,
	}

	err = backend.initialize(config)
	if err != nil {
		return nil, err
	}

	cluster := config.newCluster()
	backend.session = cluster.CreateSession()

	if config.LaunchCleanup {
		glog.Infof("Starting cleanup goroutine")
		go backend.periodicCleanup()
	}
	return backend, nil
}

// The idea behind this is simple. We calculate absolute day number
// since epoch. On even days we work with one table, on odd days we work
// with another, and periodically truncate the one that is not used at the
// moment. This function returns pair (active, inactive) of table names
func (b *CassandraBackend) tableNames() (string, string) {
	even := timeutils.EpochDay(b.UtcNow())%2 == 0
	if even {

		return "hits_even", "hits_odd"
	} else {
		return "hits_odd", "hits_even"
	}
}

func (b *CassandraBackend) nextCleanup() time.Time {
	now := b.UtcNow()
	hour := b.config.CleanupTime.Hour
	minute := b.config.CleanupTime.Minute
	nextCleanup := time.Date(
		now.Year(), now.Month(), now.Day()+1, hour, minute, 0, 0, time.UTC)
	return nextCleanup
}

// Simply loops forever and truncates the table that is not being used
func (b *CassandraBackend) periodicCleanup() {
	for {
		glog.Infof("Launching cleanup")
		b.cleanup()
		nextCleanup := b.nextCleanup()
		waitTime := nextCleanup.Sub(b.UtcNow())
		glog.Infof("Now is %s, next cleanup will happen: %s in %s",
			b.UtcNow(), nextCleanup, waitTime)
		timer := time.NewTimer(waitTime)
		select {
		case <-timer.C:
			b.cleanup()
		}
	}
}

func (b *CassandraBackend) cleanup() error {
	_, inactiveTable := b.tableNames()
	start := b.UtcNow()
	glog.Infof("Starting cleanup of: %s, time: %s", inactiveTable, start)
	err := b.session.Query(fmt.Sprintf("TRUNCATE %s", inactiveTable)).Exec()
	diff := b.UtcNow().Sub(start)
	if err != nil {
		glog.Errorf("Cleanup Failed %s", diff, err)
	} else {
		glog.Infof("Cleanup took %s and resulted in success", diff)
	}
	return err
}

// Checks if the error is actually "AlreadyExists" error which are ok
// in some cases (e.g. when we try to create keyspace or table)
func (b *CassandraBackend) isDupeError(err error) bool {
	return strings.Contains(err.Error(), "exist")
}

// Creates keyspace and tables for the counters
func (b *CassandraBackend) initialize(config *CassandraConfig) error {
	if err := b.createKeyspace(config); err != nil {
		glog.Errorf("Failed to create keyspace")
		return err
	}
	return b.createTables(config)
}

// Creates keyspace if it does not exist in a separate session
func (b *CassandraBackend) createKeyspace(config *CassandraConfig) error {
	cluster := config.newCluster()
	cluster.Keyspace = ""
	session := cluster.CreateSession()
	defer session.Close()
	err := session.Query(fmt.Sprintf(`CREATE KEYSPACE %s
				WITH replication = {
					'class' : 'SimpleStrategy',
					'replication_factor' : %d
				}`, config.Keyspace, config.ReplicationFactor)).Exec()
	if err == nil {
		return nil
	}
	if b.isDupeError(err) {
		glog.Infof("Keyspace %s already exists, it's ok", config.Keyspace)
		return nil
	} else {
		glog.Infof("Unexpected error: %s", err)
		return err
	}
}

// Creates tables for the counters in a separate session
func (b *CassandraBackend) createTables(config *CassandraConfig) error {
	cluster := config.newCluster()
	session := cluster.CreateSession()
	defer session.Close()
	active, inactive := b.tableNames()
	tables := []string{active, inactive}
	for _, tableName := range tables {
		err := session.Query(fmt.Sprintf(`CREATE TABLE %s (
                    hit text PRIMARY KEY,
                    value counter
				) WITH COMPACT STORAGE`, tableName)).Exec()
		if err != nil {
			if b.isDupeError(err) {
				glog.Infof("Table %s already exists, it's ok", tableName)
			} else {
				glog.Errorf("Unexpected error: %s", err)
				return err
			}
		}
	}
	return nil
}

func (b *CassandraBackend) UtcNow() time.Time {
	return b.timeProvider.UtcNow()
}

func (c *CassandraConfig) applyDefaults() error {
	if len(c.Servers) == 0 {
		return fmt.Errorf("At least one node is required")
	}
	if len(c.Keyspace) == 0 {
		return fmt.Errorf("Keyspace is missing")
	}
	if c.ConnectTimeout == 0 {
		c.ConnectTimeout = DefaultConnectTimeout
		glog.Infof("Setting default connection timeout: %v", c.ConnectTimeout)
	}

	if c.ReplicationFactor == 0 {
		c.ReplicationFactor = DefaultReplicationFactor
		glog.Infof("Setting default replication factor: %v", c.ReplicationFactor)
	}

	if c.CleanupTime == nil {
		c.CleanupTime = &CleanupTime{
			Hour:   DefaultCleanupHour,
			Minute: DefaultCleanupMinute,
		}
		glog.Infof("Setting cleanup time to: %v", c.CleanupTime)
	}
	return nil
}

func (c *CassandraConfig) newCluster() *gocql.ClusterConfig {
	cluster := gocql.NewCluster(c.Servers...)
	cluster.Consistency = c.Consistency
	cluster.Keyspace = c.Keyspace
	cluster.Timeout = c.ConnectTimeout
	cluster.ProtoVersion = 1
	cluster.CQLVersion = "3.0.0"
	return cluster
}
