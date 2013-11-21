package storecassandra

import (
	"github.com/cloudfoundry/hm9000/models"
	"tux21b.org/v1/gocql"
)

func (s *StoreCassandra) SyncDesiredState(newDesiredStates ...models.DesiredAppState) error {
	existingDesiredStates, err := s.GetDesiredState()
	if err != nil {
		return err
	}

	batch := s.newBatch()
	newDesiredStateKeys := make(map[string]bool, 0)
	for _, state := range newDesiredStates {
		newDesiredStateKeys[state.StoreKey()] = true
		batch.Query(`INSERT INTO DesiredStates (app_guid, app_version, number_of_instances, state, package_state) VALUES (?, ?, ?, ?, ?)`, state.AppGuid, state.AppVersion, state.NumberOfInstances, state.State, state.PackageState)
	}

	for key, existingDesiredState := range existingDesiredStates {
		if !newDesiredStateKeys[key] {
			batch.Query(`DELETE FROM DesiredStates WHERE app_guid=? AND app_version=?`, existingDesiredState.AppGuid, existingDesiredState.AppVersion)
		}
	}

	return s.session.ExecuteBatch(batch)
}

func (s *StoreCassandra) GetDesiredState() (map[string]models.DesiredAppState, error) {
	result := map[string]models.DesiredAppState{}
	desiredStates, err := s.getDesiredState("", "")

	if err != nil {
		return result, err
	}

	for _, desiredState := range desiredStates {
		result[desiredState.StoreKey()] = desiredState
	}

	return result, err
}

func (s *StoreCassandra) getDesiredState(optionalAppGuid string, optionalAppVersion string) ([]models.DesiredAppState, error) {
	result := []models.DesiredAppState{}
	var err error
	var iter *gocql.Iter

	if optionalAppGuid == "" {
		iter = s.session.Query(`SELECT app_guid, app_version, number_of_instances, state, package_state FROM DesiredStates`).Iter()
	} else {
		iter = s.session.Query(`SELECT app_guid, app_version, number_of_instances, state, package_state FROM DesiredStates WHERE app_guid = ? AND app_version = ?`, optionalAppGuid, optionalAppVersion).Iter()
	}

	var appGuid, appVersion, state, packageState string
	var numberOfInstances int32

	batch := s.newBatch()

	for iter.Scan(&appGuid, &appVersion, &numberOfInstances, &state, &packageState) {
		desiredState := models.DesiredAppState{
			AppGuid:           appGuid,
			AppVersion:        appVersion,
			NumberOfInstances: int(numberOfInstances),
			State:             models.AppState(state),
			PackageState:      models.AppPackageState(packageState),
		}
		result = append(result, desiredState)
	}

	err = iter.Close()

	if err != nil {
		return result, err
	}

	err = s.session.ExecuteBatch(batch)
	return result, err
}
