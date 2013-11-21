package storecassandra

import (
	"github.com/cloudfoundry/hm9000/store"
	"time"
	"tux21b.org/v1/gocql"
)

func (s *StoreCassandra) BumpDesiredFreshness(timestamp time.Time) error {
	return s.session.Query(`INSERT INTO Freshness (key, created_at, expires) VALUES (?, ?, ?)`, s.conf.DesiredFreshnessKey, timestamp.Unix(), timestamp.Unix()+int64(s.conf.DesiredFreshnessTTL())).Exec()
}

func (s *StoreCassandra) IsDesiredStateFresh() (bool, error) {
	var expires int64
	err := s.session.Query(`SELECT expires FROM Freshness WHERE key=?`, s.conf.DesiredFreshnessKey).Scan(&expires)

	if err == gocql.ErrNotFound {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	if expires <= s.timeProvider.Time().Unix() {
		return false, nil
	}

	return true, nil
}

func (s *StoreCassandra) BumpActualFreshness(timestamp time.Time) error {
	shouldBumpCreatedAt := false
	var expires int64
	err := s.session.Query(`SELECT expires FROM Freshness WHERE key=?`, s.conf.ActualFreshnessKey).Scan(&expires)

	if err == gocql.ErrNotFound {
		shouldBumpCreatedAt = true
	} else if err != nil {
		return err
	} else if expires <= timestamp.Unix() {
		shouldBumpCreatedAt = true
	}

	if shouldBumpCreatedAt {
		err = s.session.Query(`INSERT INTO Freshness (key, created_at) VALUES (?, ?)`, s.conf.ActualFreshnessKey, timestamp.Unix()).Exec()
		if err != nil {
			return err
		}
	}

	err = s.session.Query(`INSERT INTO Freshness (key, expires) VALUES (?, ?)`, s.conf.ActualFreshnessKey, timestamp.Unix()+int64(s.conf.ActualFreshnessTTL())).Exec()
	if err != nil {
		return err
	}

	return nil
}

func (s *StoreCassandra) IsActualStateFresh(timestamp time.Time) (bool, error) {
	var createdAt, expires int64
	err := s.session.Query(`SELECT created_at, expires FROM Freshness WHERE key=?`, s.conf.ActualFreshnessKey).Scan(&createdAt, &expires)

	if err == gocql.ErrNotFound {
		return false, nil
	}

	if err != nil {
		return false, err
	}

	currentTime := s.timeProvider.Time().Unix()

	if createdAt+int64(s.conf.ActualFreshnessTTL()) <= currentTime && currentTime < expires {
		return true, nil
	}

	return false, nil
}

func (s *StoreCassandra) VerifyFreshness(time time.Time) error {
	desiredFresh, err := s.IsDesiredStateFresh()
	if err != nil {
		return err
	}

	actualFresh, err := s.IsActualStateFresh(time)
	if err != nil {
		return err
	}

	if !desiredFresh && !actualFresh {
		return store.ActualAndDesiredAreNotFreshError
	}

	if !desiredFresh {
		return store.DesiredIsNotFreshError
	}

	if !actualFresh {
		return store.ActualIsNotFreshError
	}

	return nil
}
