package registry

import (
	"encoding/json"
	"sync"
	"time"

	mbus "github.com/cloudfoundry/go_cfmessagebus"
	"github.com/cloudfoundry/gorouter/stats"
	steno "github.com/cloudfoundry/gosteno"

	"github.com/cloudfoundry/gorouter/config"
	"github.com/cloudfoundry/gorouter/log"
	"github.com/cloudfoundry/gorouter/route"
)

type Registry struct {
	sync.RWMutex

	*steno.Logger

	*stats.ActiveApps
	*stats.TopApps

	byUri map[route.Uri]*route.Pool

	table map[tableKey]*tableEntry

	pruneStaleDropletsInterval time.Duration
	dropletStaleThreshold      time.Duration

	messageBus mbus.MessageBus

	timeOfLastUpdate time.Time
}

type tableKey struct {
	addr string
	uri  route.Uri
}

type tableEntry struct {
	endpoint  *route.Endpoint
	updatedAt time.Time
}

func NewRegistry(c *config.Config, mbus mbus.MessageBus) *Registry {
	r := &Registry{}

	r.Logger = steno.NewLogger("router.registry")

	r.ActiveApps = stats.NewActiveApps()
	r.TopApps = stats.NewTopApps()

	r.byUri = make(map[route.Uri]*route.Pool)

	r.table = make(map[tableKey]*tableEntry)

	r.pruneStaleDropletsInterval = c.PruneStaleDropletsInterval
	r.dropletStaleThreshold = c.DropletStaleThreshold

	r.messageBus = mbus

	return r
}

func (registry *Registry) Register(uri route.Uri, endpoint *route.Endpoint) {
	registry.Lock()
	defer registry.Unlock()

	uri = uri.ToLower()

	key := tableKey{
		addr: endpoint.CanonicalAddr(),
		uri:  uri,
	}

	var endpointToRegister *route.Endpoint

	entry, found := registry.table[key]
	if found {
		endpointToRegister = entry.endpoint
	} else {
		endpointToRegister = endpoint
		entry = &tableEntry{endpoint: endpoint}

		registry.table[key] = entry
	}

	pool, found := registry.byUri[uri]
	if !found {
		pool = route.NewPool()
		registry.byUri[uri] = pool
	}

	pool.Add(endpointToRegister)

	entry.updatedAt = time.Now()

	registry.timeOfLastUpdate = time.Now()
}

func (registry *Registry) Unregister(uri route.Uri, endpoint *route.Endpoint) {
	registry.Lock()
	defer registry.Unlock()

	uri = uri.ToLower()

	key := tableKey{
		addr: endpoint.CanonicalAddr(),
		uri:  uri,
	}

	registry.unregisterUri(key)
}

func (r *Registry) Lookup(uri route.Uri) (*route.Endpoint, bool) {
	r.RLock()
	defer r.RUnlock()

	pool, ok := r.lookupByUri(uri)
	if !ok {
		return nil, false
	}

	return pool.Sample()
}

func (r *Registry) LookupByPrivateInstanceId(uri route.Uri, p string) (*route.Endpoint, bool) {
	r.RLock()
	defer r.RUnlock()

	pool, ok := r.lookupByUri(uri)
	if !ok {
		return nil, false
	}

	return pool.FindByPrivateInstanceId(p)
}

func (r *Registry) lookupByUri(uri route.Uri) (*route.Pool, bool) {
	uri = uri.ToLower()
	pool, ok := r.byUri[uri]
	return pool, ok
}

func (registry *Registry) StartPruningCycle() {
	go registry.checkAndPrune()
}

func (registry *Registry) PruneStaleDroplets() {
	if registry.isStateStale() {
		log.Info("State is stale; NOT pruning")
		registry.pauseStaleTracker()
		return
	}

	registry.Lock()
	defer registry.Unlock()

	registry.pruneStaleDroplets()
}

func (r *Registry) CaptureRoutingRequest(x *route.Endpoint, t time.Time) {
	if x.ApplicationId != "" {
		r.ActiveApps.Mark(x.ApplicationId, t)
		r.TopApps.Mark(x.ApplicationId, t)
	}
}

func (registry *Registry) NumUris() int {
	registry.RLock()
	defer registry.RUnlock()

	return len(registry.byUri)
}

func (r *Registry) TimeOfLastUpdate() time.Time {
	return r.timeOfLastUpdate
}

func (r *Registry) NumEndpoints() int {
	r.RLock()
	defer r.RUnlock()

	mapForSize := make(map[string]bool)
	for _, entry := range r.table {
		mapForSize[entry.endpoint.CanonicalAddr()] = true
	}

	return len(mapForSize)
}

func (r *Registry) MarshalJSON() ([]byte, error) {
	r.RLock()
	defer r.RUnlock()

	return json.Marshal(r.byUri)
}

func (registry *Registry) isStateStale() bool {
	return !registry.messageBus.Ping()
}

func (registry *Registry) pruneStaleDroplets() {
	for key, entry := range registry.table {
		if !registry.isEntryStale(entry) {
			continue
		}

		log.Infof("Pruning stale droplet: %v, uri: %s", entry, key.uri)
		registry.unregisterUri(key)
	}
}

func (r *Registry) isEntryStale(entry *tableEntry) bool {
	return entry.updatedAt.Add(r.dropletStaleThreshold).Before(time.Now())
}

func (registry *Registry) pauseStaleTracker() {
	for _, entry := range registry.table {
		entry.updatedAt = time.Now()
	}
}

func (r *Registry) checkAndPrune() {
	if r.pruneStaleDropletsInterval == 0 {
		return
	}

	tick := time.Tick(r.pruneStaleDropletsInterval)
	for {
		select {
		case <-tick:
			log.Debug("Start to check and prune stale droplets")
			r.PruneStaleDroplets()
		}
	}
}

func (registry *Registry) unregisterUri(key tableKey) {
	entry, found := registry.table[key]
	if !found {
		return
	}

	endpoints, found := registry.byUri[key.uri]
	if found {
		endpoints.Remove(entry.endpoint)

		if endpoints.IsEmpty() {
			delete(registry.byUri, key.uri)
		}
	}

	delete(registry.table, key)
}
