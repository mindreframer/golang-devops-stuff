package store

import (
	"fmt"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"time"
)

func (store *RealStore) crashCountStoreKey(crashCount models.CrashCount) string {
	return "/apps/" + store.AppKey(crashCount.AppGuid, crashCount.AppVersion) + "/crashes/" + crashCount.StoreKey()
}

func (store *RealStore) SaveCrashCounts(crashCounts ...models.CrashCount) error {
	t := time.Now()

	nodes := make([]storeadapter.StoreNode, len(crashCounts))
	for i, crashCount := range crashCounts {
		nodes[i] = storeadapter.StoreNode{
			Key:   store.crashCountStoreKey(crashCount),
			Value: crashCount.ToJSON(),
			TTL:   uint64(store.config.MaximumBackoffDelay().Seconds()) * 2,
		}
	}

	err := store.adapter.Set(nodes)

	store.logger.Info(fmt.Sprintf("Save Duration Crash Counts"), map[string]string{
		"Number of Items": fmt.Sprintf("%d", len(crashCounts)),
		"Duration":        fmt.Sprintf("%.4f seconds", time.Since(t).Seconds()),
	})
	return err
}
