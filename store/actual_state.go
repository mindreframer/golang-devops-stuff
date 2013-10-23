package store

import (
	"fmt"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"time"
)

func (store *RealStore) actualStateStoreKey(actualState models.InstanceHeartbeat) string {
	return "/apps/" + store.AppKey(actualState.AppGuid, actualState.AppVersion) + "/actual/" + actualState.StoreKey()
}

func (store *RealStore) SaveActualState(actualStates ...models.InstanceHeartbeat) error {
	t := time.Now()

	nodes := make([]storeadapter.StoreNode, len(actualStates))
	for i, actualState := range actualStates {
		nodes[i] = storeadapter.StoreNode{
			Key:   store.actualStateStoreKey(actualState),
			Value: actualState.ToJSON(),
			TTL:   store.config.HeartbeatTTL(),
		}
	}

	err := store.adapter.Set(nodes)

	store.logger.Info(fmt.Sprintf("Save Duration Actual"), map[string]string{
		"Number of Items": fmt.Sprintf("%d", len(actualStates)),
		"Duration":        fmt.Sprintf("%.4f seconds", time.Since(t).Seconds()),
	})
	return err
}
