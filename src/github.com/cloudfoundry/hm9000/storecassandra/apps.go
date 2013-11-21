package storecassandra

import (
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/store"
)

func (s *StoreCassandra) AppKey(appGuid string, appVersion string) string {
	return appGuid + "-" + appVersion
}

func (s *StoreCassandra) GetApps() (map[string]*models.App, error) {
	apps := map[string]*models.App{}

	desiredStates, err := s.GetDesiredState()
	if err != nil {
		return apps, err
	}

	actualStates, err := s.GetInstanceHeartbeats()
	if err != nil {
		return apps, err
	}

	crashCounts, err := s.getCrashCounts("", "")
	if err != nil {
		return apps, err
	}

	for _, desiredState := range desiredStates {
		key := s.AppKey(desiredState.AppGuid, desiredState.AppVersion)
		apps[key] = models.NewApp(desiredState.AppGuid, desiredState.AppVersion, desiredState, []models.InstanceHeartbeat{}, map[int]models.CrashCount{})
	}

	for _, actualState := range actualStates {
		key := s.AppKey(actualState.AppGuid, actualState.AppVersion)
		app, found := apps[key]

		if found {
			app.InstanceHeartbeats = append(app.InstanceHeartbeats, actualState)
		} else {
			apps[key] = models.NewApp(actualState.AppGuid, actualState.AppVersion, models.DesiredAppState{}, []models.InstanceHeartbeat{actualState}, map[int]models.CrashCount{})
		}
	}

	for _, crashCount := range crashCounts {
		key := s.AppKey(crashCount.AppGuid, crashCount.AppVersion)
		app, found := apps[key]

		if found {
			app.CrashCounts[crashCount.InstanceIndex] = crashCount
		}
	}

	return apps, nil
}

func (s *StoreCassandra) GetApp(appGuid string, appVersion string) (*models.App, error) {
	desiredState, err := s.getDesiredStateForApp(appGuid, appVersion)
	if err != nil {
		return nil, err
	}

	actualStates, err := s.getActualStatesForApp(appGuid, appVersion)
	if err != nil {
		return nil, err
	}

	if desiredState.AppGuid == "" && len(actualStates) == 0 {
		return nil, store.AppNotFoundError
	}

	crashCounts, err := s.getCrashCountsForApp(appGuid, appVersion)
	if err != nil {
		return nil, err
	}

	return models.NewApp(appGuid, appVersion, desiredState, actualStates, crashCounts), nil
}

func (s *StoreCassandra) getDesiredStateForApp(appGuid string, appVersion string) (models.DesiredAppState, error) {
	desiredStates, err := s.getDesiredState(appGuid, appVersion)

	if err != nil {
		return models.DesiredAppState{}, err
	}

	if len(desiredStates) == 0 {
		return models.DesiredAppState{}, nil
	} else {
		return desiredStates[0], nil
	}
}

func (s *StoreCassandra) getActualStatesForApp(appGuid string, appVersion string) ([]models.InstanceHeartbeat, error) {
	return s.getActualState(appGuid, appVersion)
}

func (s *StoreCassandra) getCrashCountsForApp(appGuid string, appVersion string) (map[int]models.CrashCount, error) {
	crashCounts, err := s.getCrashCounts(appGuid, appVersion)
	result := map[int]models.CrashCount{}

	if err != nil {
		return result, err
	}

	for _, crashCount := range crashCounts {
		result[crashCount.InstanceIndex] = crashCount
	}

	return result, err
}
