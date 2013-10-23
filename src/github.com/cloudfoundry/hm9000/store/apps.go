package store

import (
	"fmt"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"strings"
	"time"
)

func (store *RealStore) AppKey(appGuid string, appVersion string) string {
	return appGuid + "-" + appVersion
}

func (store *RealStore) GetApp(appGuid string, appVersion string) (*models.App, error) {
	t := time.Now()
	node, err := store.adapter.ListRecursively("/apps/" + store.AppKey(appGuid, appVersion))

	if err == storeadapter.ErrorKeyNotFound {
		return nil, AppNotFoundError
	} else if err != nil {
		return nil, err
	}

	app, err := store.appNodeToApp(node)
	if app == nil {
		return nil, AppNotFoundError
	}

	store.logger.Info(fmt.Sprintf("Get Duration App"), map[string]string{
		"Duration": fmt.Sprintf("%.4f seconds", time.Since(t).Seconds()),
	})

	return app, err
}

func (store *RealStore) GetApps() (results map[string]*models.App, err error) {
	t := time.Now()

	results = make(map[string]*models.App)

	node, err := store.adapter.ListRecursively("/apps")

	if err == storeadapter.ErrorKeyNotFound {
		return results, nil
	} else if err != nil {
		return results, err
	}

	for _, appNode := range node.ChildNodes {
		app, err := store.appNodeToApp(appNode)
		if err != nil {
			return make(map[string]*models.App), err
		}
		if app != nil {
			results[store.AppKey(app.AppGuid, app.AppVersion)] = app
		}
	}

	store.logger.Info(fmt.Sprintf("Get Duration Apps"), map[string]string{
		"Number of Items": fmt.Sprintf("%d", len(results)),
		"Duration":        fmt.Sprintf("%.4f seconds", time.Since(t).Seconds()),
	})

	return results, nil
}

func (store *RealStore) appNodeToApp(appNode storeadapter.StoreNode) (*models.App, error) {
	desiredState := models.DesiredAppState{}
	actualState := []models.InstanceHeartbeat{}
	crashCounts := make(map[int]models.CrashCount)

	appGuid := ""
	appVersion := ""

	for _, childNode := range appNode.ChildNodes {
		if strings.HasSuffix(childNode.Key, "desired") {
			desired, err := models.NewDesiredAppStateFromJSON(childNode.Value)
			if err != nil {
				return nil, err
			}
			desiredState = desired
			appGuid = desired.AppGuid
			appVersion = desired.AppVersion
		} else if strings.HasSuffix(childNode.Key, "actual") {
			for _, actualNode := range childNode.ChildNodes {
				actual, err := models.NewInstanceHeartbeatFromJSON(actualNode.Value)
				if err != nil {
					return nil, err
				}
				actualState = append(actualState, actual)
				appGuid = actual.AppGuid
				appVersion = actual.AppVersion
			}
		} else if strings.HasSuffix(childNode.Key, "crashes") {
			for _, crashNode := range childNode.ChildNodes {
				crashCount, err := models.NewCrashCountFromJSON(crashNode.Value)
				if err != nil {
					return nil, err
				}
				crashCounts[crashCount.InstanceIndex] = crashCount
			}
		}
	}

	if appGuid == "" || appVersion == "" {
		return nil, nil
	}

	return models.NewApp(appGuid, appVersion, desiredState, actualState, crashCounts), nil
}
