package store

import (
	"fmt"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/storeadapter"
	"strings"
	"time"
)

func (store *RealStore) ensureCacheIsReady() error {
	store.instanceHeartbeatCacheMutex.Lock()
	defer store.instanceHeartbeatCacheMutex.Unlock()

	if time.Since(store.instanceHeartbeatCacheTimestamp) >= store.config.StoreHeartbeatCacheRefreshInterval() {
		t := time.Now()
		heartbeats, err := store.GetInstanceHeartbeats()
		if err != nil {
			return err
		}

		store.instanceHeartbeatCache = map[string]models.InstanceHeartbeat{}
		for _, heartbeat := range heartbeats {
			store.instanceHeartbeatCache[heartbeat.InstanceGuid] = heartbeat
		}
		store.instanceHeartbeatCacheTimestamp = time.Now()
		store.logger.Debug("Busting store cache", map[string]string{
			"Duration":                   time.Since(t).String(),
			"Instance Heartbeats Loaded": fmt.Sprintf("%d", len(store.instanceHeartbeatCache)),
		})

	}

	return nil
}

func (store *RealStore) SyncHeartbeats(incomingHeartbeats ...models.Heartbeat) error {
	t := time.Now()

	err := store.ensureCacheIsReady()
	if err != nil {
		return err
	}

	nodesToSave := []storeadapter.StoreNode{}
	keysToDelete := []string{}
	numberOfInstanceHeartbeats := 0

	store.instanceHeartbeatCacheMutex.Lock()

	for _, incomingHeartbeat := range incomingHeartbeats {
		numberOfInstanceHeartbeats += len(incomingHeartbeat.InstanceHeartbeats)
		incomingInstanceGuids := map[string]bool{}
		nodesToSave = append(nodesToSave, store.deaPresenceNode(incomingHeartbeat.DeaGuid))
		for _, incomingInstanceHeartbeat := range incomingHeartbeat.InstanceHeartbeats {
			incomingInstanceGuids[incomingInstanceHeartbeat.InstanceGuid] = true
			existingInstanceHeartbeat, found := store.instanceHeartbeatCache[incomingInstanceHeartbeat.InstanceGuid]

			if found && existingInstanceHeartbeat.State == incomingInstanceHeartbeat.State {
				continue
			}

			nodesToSave = append(nodesToSave, store.storeNodeForInstanceHeartbeat(incomingInstanceHeartbeat))
			store.instanceHeartbeatCache[incomingInstanceHeartbeat.InstanceGuid] = incomingInstanceHeartbeat
		}

		cacheKeysToDelete := []string{}

		for _, existingInstanceHeartbeat := range store.instanceHeartbeatCache {
			if existingInstanceHeartbeat.DeaGuid == incomingHeartbeat.DeaGuid && !incomingInstanceGuids[existingInstanceHeartbeat.InstanceGuid] {
				key := store.instanceHeartbeatStoreKey(existingInstanceHeartbeat.AppGuid, existingInstanceHeartbeat.AppVersion, existingInstanceHeartbeat.InstanceGuid)
				keysToDelete = append(keysToDelete, key)
				cacheKeysToDelete = append(cacheKeysToDelete, existingInstanceHeartbeat.InstanceGuid)
			}
		}

		for _, key := range cacheKeysToDelete {
			delete(store.instanceHeartbeatCache, key)
		}
	}

	store.instanceHeartbeatCacheMutex.Unlock()

	tSave := time.Now()
	err = store.adapter.SetMulti(nodesToSave)
	dtSave := time.Since(tSave).Seconds()

	if err != nil {
		return err
	}

	tDelete := time.Now()
	err = store.adapter.Delete(keysToDelete...)
	dtDelete := time.Since(tDelete).Seconds()

	if err == storeadapter.ErrorKeyNotFound {
		store.logger.Debug("store.SyncHeartbeats Failed to delete a key, soldiering on...")
	} else if err != nil {
		return err
	}

	store.logger.Debug(fmt.Sprintf("Save Duration Actual"), map[string]string{
		"Number of Heartbeats":          fmt.Sprintf("%d", len(incomingHeartbeats)),
		"Number of Instance Heartbeats": fmt.Sprintf("%d", numberOfInstanceHeartbeats),
		"Number of Items Saved":         fmt.Sprintf("%d", len(nodesToSave)),
		"Number of Items Deleted":       fmt.Sprintf("%d", len(keysToDelete)),
		"Duration":                      fmt.Sprintf("%.4f seconds", time.Since(t).Seconds()),
		"Save Duration":                 fmt.Sprintf("%.4f seconds", dtSave),
		"Delete Duration":               fmt.Sprintf("%.4f seconds", dtDelete),
	})

	return nil
}

func (store *RealStore) GetInstanceHeartbeats() (results []models.InstanceHeartbeat, err error) {
	results = []models.InstanceHeartbeat{}
	node, err := store.adapter.ListRecursively(store.SchemaRoot() + "/apps/actual")
	if err == storeadapter.ErrorKeyNotFound {
		return results, nil
	} else if err != nil {
		return results, err
	}

	unexpiredDeas, err := store.unexpiredDeas()
	if err != nil {
		return results, err
	}

	expiredKeys := []string{}
	for _, actualNode := range node.ChildNodes {
		heartbeats, toDelete, err := store.heartbeatsForNode(actualNode, unexpiredDeas)
		if err != nil {
			return []models.InstanceHeartbeat{}, nil
		}
		results = append(results, heartbeats...)
		expiredKeys = append(expiredKeys, toDelete...)
	}

	err = store.adapter.Delete(expiredKeys...)
	if err == storeadapter.ErrorKeyNotFound {
		store.logger.Debug("store.GetInstanceHeartbeats Failed to delete a key, soldiering on...")
	} else if err != nil {
		return []models.InstanceHeartbeat{}, err
	}

	return results, nil
}

func (store *RealStore) GetInstanceHeartbeatsForApp(appGuid string, appVersion string) (results []models.InstanceHeartbeat, err error) {
	node, err := store.adapter.ListRecursively(store.SchemaRoot() + "/apps/actual/" + store.AppKey(appGuid, appVersion))
	if err == storeadapter.ErrorKeyNotFound {
		return []models.InstanceHeartbeat{}, nil
	} else if err != nil {
		return []models.InstanceHeartbeat{}, err
	}

	unexpiredDeas, err := store.unexpiredDeas()
	if err != nil {
		return results, err
	}

	results, expiredKeys, err := store.heartbeatsForNode(node, unexpiredDeas)
	if err != nil {
		return []models.InstanceHeartbeat{}, err
	}

	err = store.adapter.Delete(expiredKeys...)
	if err == storeadapter.ErrorKeyNotFound {
		store.logger.Debug("store.GetInstanceHeartbeatsForApp Failed to delete a key, soldiering on...")
	} else if err != nil {
		return []models.InstanceHeartbeat{}, err
	}

	return results, nil
}

func (store *RealStore) heartbeatsForNode(node storeadapter.StoreNode, unexpiredDeas map[string]bool) (results []models.InstanceHeartbeat, toDelete []string, err error) {
	results = []models.InstanceHeartbeat{}
	for _, heartbeatNode := range node.ChildNodes {
		components := strings.Split(heartbeatNode.Key, "/")
		instanceGuid := components[len(components)-1]
		appGuidVersion := strings.Split(components[len(components)-2], ",")
		heartbeat, err := models.NewInstanceHeartbeatFromCSV(appGuidVersion[0], appGuidVersion[1], instanceGuid, heartbeatNode.Value)
		if err != nil {
			return []models.InstanceHeartbeat{}, []string{}, err
		}

		_, deaIsPresent := unexpiredDeas[heartbeat.DeaGuid]

		if deaIsPresent {
			results = append(results, heartbeat)
		} else {
			toDelete = append(toDelete, heartbeatNode.Key)
		}
	}

	return results, toDelete, nil
}

func (store *RealStore) unexpiredDeas() (results map[string]bool, err error) {
	results = map[string]bool{}

	summaryNodes, err := store.adapter.ListRecursively(store.SchemaRoot() + "/dea-presence")
	if err == storeadapter.ErrorKeyNotFound {
		return results, nil
	} else if err != nil {
		return results, err
	}

	for _, deaPresenceNode := range summaryNodes.ChildNodes {
		results[string(deaPresenceNode.Value)] = true
	}

	return results, nil
}

func (store *RealStore) instanceHeartbeatStoreKey(appGuid string, appVersion string, instanceGuid string) string {
	return store.SchemaRoot() + "/apps/actual/" + store.AppKey(appGuid, appVersion) + "/" + instanceGuid
}

func (store *RealStore) deaPresenceNode(deaGuid string) storeadapter.StoreNode {
	return storeadapter.StoreNode{
		Key:   store.SchemaRoot() + "/dea-presence/" + deaGuid,
		Value: []byte(deaGuid),
		TTL:   store.config.HeartbeatTTL(),
	}
}

func (store *RealStore) storeNodeForInstanceHeartbeat(instanceHeartbeat models.InstanceHeartbeat) storeadapter.StoreNode {
	return storeadapter.StoreNode{
		Key:   store.instanceHeartbeatStoreKey(instanceHeartbeat.AppGuid, instanceHeartbeat.AppVersion, instanceHeartbeat.InstanceGuid),
		Value: instanceHeartbeat.ToCSV(),
	}
}
