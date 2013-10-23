package store

import (
	"fmt"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"time"
)

func (store *RealStore) desiredStateStoreKey(desiredState models.DesiredAppState) string {
	return "/apps/" + store.AppKey(desiredState.AppGuid, desiredState.AppVersion) + "/desired"
}

func (store *RealStore) SaveDesiredState(desiredStates ...models.DesiredAppState) error {
	t := time.Now()

	nodes := make([]storeadapter.StoreNode, len(desiredStates))
	for i, desiredState := range desiredStates {
		nodes[i] = storeadapter.StoreNode{
			Key:   store.desiredStateStoreKey(desiredState),
			Value: desiredState.ToJSON(),
			TTL:   store.config.DesiredStateTTL(),
		}
	}

	err := store.adapter.Set(nodes)

	store.logger.Info(fmt.Sprintf("Save Duration Desired"), map[string]string{
		"Number of Items": fmt.Sprintf("%d", len(desiredStates)),
		"Duration":        fmt.Sprintf("%.4f seconds", time.Since(t).Seconds()),
	})
	return err
}

func (store *RealStore) GetDesiredState() (results map[string]models.DesiredAppState, err error) {
	t := time.Now()

	results = make(map[string]models.DesiredAppState)

	apps, err := store.GetApps()
	if err != nil {
		return results, err
	}

	for _, app := range apps {
		if app.Desired.AppGuid != "" {
			results[app.Desired.StoreKey()] = app.Desired
		}
	}

	store.logger.Info(fmt.Sprintf("Get Duration Desired"), map[string]string{
		"Number of Items": fmt.Sprintf("%d", len(results)),
		"Duration":        fmt.Sprintf("%.4f seconds", time.Since(t).Seconds()),
	})
	return results, nil
}

func (store *RealStore) DeleteDesiredState(desiredStates ...models.DesiredAppState) error {
	t := time.Now()

	for _, desiredState := range desiredStates {
		err := store.adapter.Delete(store.desiredStateStoreKey(desiredState))
		if err != nil {
			return err
		}
	}

	store.logger.Info(fmt.Sprintf("Delete Duration Desired"), map[string]string{
		"Number of Items": fmt.Sprintf("%d", len(desiredStates)),
		"Duration":        fmt.Sprintf("%.4f seconds", time.Since(t).Seconds()),
	})

	return nil
}
