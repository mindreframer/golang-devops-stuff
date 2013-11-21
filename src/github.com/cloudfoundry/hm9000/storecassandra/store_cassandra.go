package storecassandra

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/timeprovider"
	"time"
	"tux21b.org/v1/gocql"
)

type StoreCassandra struct {
	clusterConfig *gocql.ClusterConfig
	session       *gocql.Session
	conf          config.Config
	timeProvider  timeprovider.TimeProvider
	consistency   gocql.Consistency
}

func New(clusterURLs []string, consistency gocql.Consistency, conf config.Config, timeProvider timeprovider.TimeProvider) (*StoreCassandra, error) {
	s := &StoreCassandra{
		conf:         conf,
		timeProvider: timeProvider,
		consistency:  consistency,
	}

	s.clusterConfig = gocql.NewCluster(clusterURLs...)
	s.clusterConfig.Consistency = s.consistency
	s.clusterConfig.Timeout = 5 * time.Second

	var err error

	err = s.createKeySpace()
	if err != nil {
		return s, err
	}

	s.clusterConfig.Keyspace = "hm9000"
	s.session, err = s.clusterConfig.CreateSession()
	if err != nil {
		return s, err
	}

	err = s.createTables()
	if err != nil {
		return s, err
	}

	return s, nil
}

func (s *StoreCassandra) createKeySpace() error {
	var err error
	s.session, err = s.clusterConfig.CreateSession()
	if err != nil {
		return err
	}

	err = s.session.Query(`CREATE KEYSPACE IF NOT EXISTS hm9000 WITH REPLICATION = { 'class' : 'SimpleStrategy', 'replication_factor' : 1 }`).Exec()
	if err != nil {
		return err
	}
	s.session.Close()

	return nil
}

func (s *StoreCassandra) createTables() error {
	var err error
	err = s.session.Query(`CREATE TABLE IF NOT EXISTS DesiredStates (app_guid text, app_version text, number_of_instances int, state text, package_state text, PRIMARY KEY (app_guid, app_version))`).Exec()
	if err != nil {
		return err
	}

	err = s.session.Query(`CREATE TABLE IF NOT EXISTS ActualStates (app_guid text, app_version text, instance_guid text, dea_guid text, instance_index int, state text, state_timestamp bigint, expires bigint, PRIMARY KEY (app_guid, app_version, instance_guid))`).Exec()
	if err != nil {
		return err
	}

	err = s.session.Query(`CREATE INDEX IF NOT EXISTS dea_guid_key on ActualStates (dea_guid)`).Exec()
	if err != nil {
		return err
	}

	err = s.session.Query(`CREATE TABLE IF NOT EXISTS CrashCounts (app_guid text, app_version text, instance_index int, crash_count int, created_at bigint, expires bigint, PRIMARY KEY (app_guid, app_version, instance_index))`).Exec()
	if err != nil {
		return err
	}

	err = s.session.Query(`CREATE TABLE IF NOT EXISTS PendingStartMessages (app_guid text, app_version text, message_id text, send_on bigint, sent_on bigint, keep_alive int, index_to_start int, priority double, skip_verification boolean, reason text, PRIMARY KEY (app_guid, app_version, index_to_start))`).Exec()
	if err != nil {
		return err
	}

	err = s.session.Query(`CREATE TABLE IF NOT EXISTS PendingStopMessages (app_guid text, app_version text, message_id text, send_on bigint, sent_on bigint, keep_alive int, instance_guid text, reason text, PRIMARY KEY (app_guid, app_version, instance_guid))`).Exec()
	if err != nil {
		return err
	}

	err = s.session.Query(`CREATE TABLE IF NOT EXISTS Freshness (key text, created_at bigint, expires bigint, PRIMARY KEY (key))`).Exec()
	if err != nil {
		return err
	}

	err = s.session.Query(`CREATE TABLE IF NOT EXISTS Metrics (key text, value double, PRIMARY KEY (key))`).Exec()
	if err != nil {
		return err
	}

	return err

}

func (s *StoreCassandra) newBatch() *gocql.Batch {
	batch := gocql.NewBatch(gocql.LoggedBatch)
	batch.Cons = s.consistency
	return batch
}

func (s *StoreCassandra) Compact() error {
	return nil
}
