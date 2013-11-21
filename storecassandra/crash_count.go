package storecassandra

import (
	"github.com/cloudfoundry/hm9000/models"
	"tux21b.org/v1/gocql"
)

func (s *StoreCassandra) SaveCrashCounts(crashCounts ...models.CrashCount) error {
	batch := s.newBatch()

	for _, crashCount := range crashCounts {
		batch.Query(`INSERT INTO CrashCounts (app_guid, app_version, instance_index, crash_count, created_at, expires) VALUES (?, ?, ?, ?, ?, ?)`,
			crashCount.AppGuid,
			crashCount.AppVersion,
			int32(crashCount.InstanceIndex),
			int32(crashCount.CrashCount),
			crashCount.CreatedAt,
			s.timeProvider.Time().Unix()+int64(s.conf.MaximumBackoffDelay().Seconds()*2))
	}

	return s.session.ExecuteBatch(batch)
}

func (s *StoreCassandra) GetCrashCounts() (map[string]models.CrashCount, error) {
	crashCounts, err := s.getCrashCounts("", "")
	result := map[string]models.CrashCount{}

	if err != nil {
		return result, err
	}

	for _, crashCount := range crashCounts {
		result[crashCount.StoreKey()] = crashCount
	}

	return result, err
}

func (s *StoreCassandra) getCrashCounts(optionalAppGuid string, optionalAppVersion string) ([]models.CrashCount, error) {
	result := []models.CrashCount{}
	var err error

	var iter *gocql.Iter
	if optionalAppGuid != "" {
		iter = s.session.Query(`SELECT app_guid, app_version, instance_index, crash_count, created_at, expires FROM CrashCounts WHERE app_guid = ? AND app_version = ?`, optionalAppGuid, optionalAppVersion).Iter()
	} else {
		iter = s.session.Query(`SELECT app_guid, app_version, instance_index, crash_count, created_at, expires FROM CrashCounts`).Iter()
	}

	var appGuid, appVersion string
	var instanceIndex, crashCount int32
	var createdAt, expires int64

	currentTime := s.timeProvider.Time().Unix()

	batch := s.newBatch()

	for iter.Scan(&appGuid, &appVersion, &instanceIndex, &crashCount, &createdAt, &expires) {
		if expires <= currentTime {
			batch.Query(`DELETE FROM CrashCounts WHERE app_guid=? AND app_version=? AND instance_index = ?`, appGuid, appVersion, instanceIndex)
		} else {
			crashCount := models.CrashCount{
				AppGuid:       appGuid,
				AppVersion:    appVersion,
				InstanceIndex: int(instanceIndex),
				CrashCount:    int(crashCount),
				CreatedAt:     createdAt,
			}
			result = append(result, crashCount)
		}
	}

	err = iter.Close()

	if err != nil {
		return result, err
	}

	err = s.session.ExecuteBatch(batch)

	return result, err
}
