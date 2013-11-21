package store

import (
	"fmt"
	"github.com/cloudfoundry/hm9000/models"
	"time"
)

func (store *RealStore) AppKey(appGuid string, appVersion string) string {
	return appGuid + "," + appVersion
}

func (store *RealStore) GetApp(appGuid string, appVersion string) (*models.App, error) {
	t := time.Now()

	representation := &appRepresentation{
		actualState: []models.InstanceHeartbeat{},
		crashCounts: []models.CrashCount{},
	}

	var err error

	tDesired := time.Now()
	representation.desiredState, err = store.getDesiredStateForApp(appGuid, appVersion)
	if err != nil {
		return nil, err
	}
	dtDesired := time.Since(tDesired).Seconds()

	tActual := time.Now()
	representation.actualState, err = store.GetInstanceHeartbeatsForApp(appGuid, appVersion)
	if err != nil {
		return nil, err
	}
	dtActual := time.Since(tActual).Seconds()

	if !representation.representsAnApp() {
		return nil, AppNotFoundError
	}

	tCrash := time.Now()
	representation.crashCounts, err = store.getCrashCountForApp(appGuid, appVersion)
	if err != nil {
		return nil, err
	}
	dtCrash := time.Since(tCrash).Seconds()

	app, err := representation.buildApp()
	if app == nil {
		return nil, AppNotFoundError
	}

	store.logger.Debug(fmt.Sprintf("Get Duration App"), map[string]string{
		"Duration":                   fmt.Sprintf("%.4f seconds", time.Since(t).Seconds()),
		"Time to Fetch Desired":      fmt.Sprintf("%.4f seconds", dtDesired),
		"Time to Fetch Actual":       fmt.Sprintf("%.4f seconds", dtActual),
		"Time to Fetch Crash Counts": fmt.Sprintf("%.4f seconds", dtCrash),
	})

	return app, err
}

func (store *RealStore) GetApps() (results map[string]*models.App, err error) {
	t := time.Now()

	results = make(map[string]*models.App)
	representations := make(appRepresentations)

	tDesired := time.Now()
	desiredStates, err := store.GetDesiredState()
	dtDesired := time.Since(tDesired).Seconds()

	if err != nil {
		return results, err
	}
	for _, desiredState := range desiredStates {
		representation := representations.representationForAppGuidVersion(desiredState.AppGuid, desiredState.AppVersion)
		representation.desiredState = desiredState
	}

	tActual := time.Now()
	actualStates, err := store.GetInstanceHeartbeats()
	dtActual := time.Since(tActual).Seconds()
	if err != nil {
		return results, err
	}
	for _, actualState := range actualStates {
		representation := representations.representationForAppGuidVersion(actualState.AppGuid, actualState.AppVersion)
		representation.actualState = append(representation.actualState, actualState)
	}

	tCrash := time.Now()
	crashCounts, err := store.getCrashCounts()
	dtCrash := time.Since(tCrash).Seconds()

	if err != nil {
		return results, err
	}
	for _, crashCount := range crashCounts {
		representation := representations.representationForAppGuidVersion(crashCount.AppGuid, crashCount.AppVersion)
		representation.crashCounts = append(representation.crashCounts, crashCount)
	}

	for _, appRepresentation := range representations {
		if appRepresentation.representsAnApp() {
			app, err := appRepresentation.buildApp()
			if err != nil {
				return make(map[string]*models.App), err
			}
			if app != nil {
				results[store.AppKey(app.AppGuid, app.AppVersion)] = app
			}
		}
	}

	store.logger.Debug(fmt.Sprintf("Get Duration Apps"), map[string]string{
		"Number of Items":            fmt.Sprintf("%d", len(results)),
		"Duration":                   fmt.Sprintf("%.4f seconds", time.Since(t).Seconds()),
		"Time to Fetch Desired":      fmt.Sprintf("%.4f seconds", dtDesired),
		"Time to Fetch Actual":       fmt.Sprintf("%.4f seconds", dtActual),
		"Time to Fetch Crash Counts": fmt.Sprintf("%.4f seconds", dtCrash),
	})

	return results, nil
}

type appRepresentations map[string]*appRepresentation

func (representations appRepresentations) representationForAppGuidVersion(appGuid string, appVersion string) *appRepresentation {
	id := appGuid + "-" + appVersion
	_, exists := representations[id]
	if !exists {
		representations[id] = &appRepresentation{
			actualState: []models.InstanceHeartbeat{},
			crashCounts: []models.CrashCount{},
		}
	}
	return representations[id]
}

type appRepresentation struct {
	desiredState models.DesiredAppState
	actualState  []models.InstanceHeartbeat
	crashCounts  []models.CrashCount
}

func (representation *appRepresentation) hasDesired() bool {
	return representation.desiredState.AppGuid != ""
}

func (representation *appRepresentation) representsAnApp() bool {
	return representation.hasDesired() || len(representation.actualState) > 0
}

func (representation *appRepresentation) buildApp() (*models.App, error) {
	appGuid := ""
	appVersion := ""

	desiredState := models.DesiredAppState{}
	if representation.hasDesired() {
		desiredState = representation.desiredState
		appGuid = desiredState.AppGuid
		appVersion = desiredState.AppVersion
	}

	actualState := representation.actualState
	if len(actualState) > 0 {
		appGuid = actualState[0].AppGuid
		appVersion = actualState[0].AppVersion
	}

	if appGuid == "" || appVersion == "" {
		return nil, nil
	}

	crashCounts := make(map[int]models.CrashCount)
	for _, crashCount := range representation.crashCounts {
		crashCounts[crashCount.InstanceIndex] = crashCount
	}

	return models.NewApp(appGuid, appVersion, desiredState, actualState, crashCounts), nil
}
