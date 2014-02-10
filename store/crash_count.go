package store

import (
	"fmt"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/storeadapter"
	"strconv"
	"time"
)

func (store *RealStore) crashCountStoreKey(crashCount models.CrashCount) string {
	return store.SchemaRoot() + "/apps/crashes/" + store.AppKey(crashCount.AppGuid, crashCount.AppVersion) + "/" + strconv.Itoa(crashCount.InstanceIndex)
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

	err := store.adapter.SetMulti(nodes)

	store.logger.Debug(fmt.Sprintf("Save Duration Crash Counts"), map[string]string{
		"Number of Items": fmt.Sprintf("%d", len(crashCounts)),
		"Duration":        fmt.Sprintf("%.4f seconds", time.Since(t).Seconds()),
	})
	return err
}

func (store *RealStore) getCrashCounts() (results []models.CrashCount, err error) {
	node, err := store.adapter.ListRecursively(store.SchemaRoot() + "/apps/crashes")

	if err == storeadapter.ErrorKeyNotFound {
		return results, nil
	} else if err != nil {
		return results, err
	}

	for _, crashNode := range node.ChildNodes {
		crashCounts, err := store.crashCountsForNode(crashNode)
		if err != nil {
			return []models.CrashCount{}, nil
		}
		results = append(results, crashCounts...)
	}

	return results, nil
}

func (store *RealStore) getCrashCountForApp(appGuid string, appVersion string) (results []models.CrashCount, err error) {
	node, err := store.adapter.ListRecursively(store.SchemaRoot() + "/apps/crashes/" + store.AppKey(appGuid, appVersion))
	if err == storeadapter.ErrorKeyNotFound {
		return []models.CrashCount{}, nil
	} else if err != nil {
		return []models.CrashCount{}, err
	}

	return store.crashCountsForNode(node)
}

func (store *RealStore) crashCountsForNode(node storeadapter.StoreNode) (results []models.CrashCount, err error) {
	for _, crashNode := range node.ChildNodes {
		crashCount, err := models.NewCrashCountFromJSON(crashNode.Value)
		if err != nil {
			return []models.CrashCount{}, err
		}

		results = append(results, crashCount)
	}
	return results, nil
}
