package roundrobin

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/mailgun/minheap"
	"github.com/mailgun/vulcan/loadbalance"
	"hash/fnv"
)

// This structure holds cursors for group of endpoints, so we can keep state
// between iterations. Unused cursors are garbage collected if not used for
// some certain period of time.
type cursorMap struct {
	// collection of cursors identified by the ids of endpoints
	cursors map[uint32][]*cursor
	// keep expiration times in the priority queue (min heap) so we can TTL effectively
	// priority queue holds the ttls
	expiryTimes *minheap.MinHeap
}

func newCursorMap() *cursorMap {
	return &cursorMap{
		cursors:     make(map[uint32][]*cursor),
		expiryTimes: minheap.NewMinHeap(),
	}
}

// Creates a new cursor and returns the existing one if it already exists. If the cursor exists,
// function updates it ttl so it won't expire in the nearest future.
func (cm *cursorMap) upsertCursor(endpoints []loadbalance.Endpoint, expiryTime int) *cursor {
	c := cm.getCursor(endpoints)
	if c != nil {
		// In case if the set is present, use it and update expiry seconds
		cm.expiryTimes.UpdateEl(c.item, expiryTime)
		return c
	} else {
		c := cm.addCursor(endpoints)
		// In case if we have not seen this set of endpoints before,
		// add it to the expiryTimes priority queue and the map of our endpoint set
		item := &minheap.Element{
			Value:    c,
			Priority: expiryTime,
		}
		c.item = item
		cm.expiryTimes.PushEl(item)
		return c
	}
}

// Returns cursor for the given endpoints set, or returns nil if there's no such cursor
func (cm *cursorMap) getCursor(endpoints []loadbalance.Endpoint) *cursor {
	cursorHash := computeHash(endpoints)
	// Find if the endpoints combination we are referring to already exists
	cursors, exists := cm.cursors[cursorHash]
	if !exists {
		return nil
	}
	if len(cursors) == 1 {
		return cursors[0]
	} else {
		for _, c := range cursors {
			if c.sameEndpoints(endpoints) {
				return c
			}
		}
	}
	return nil
}

// Add a new cursor to the collection of cursors, handles collisions by appending to the slice of cursors
func (cm *cursorMap) addCursor(endpoints []loadbalance.Endpoint) *cursor {
	c := newCursor(endpoints)
	cursors, exists := cm.cursors[c.hash]
	if !exists {
		cm.cursors[c.hash] = []*cursor{c}
	} else {
		cm.cursors[c.hash] = append(cursors, c)
		glog.Infof("RoundRobin: WOW collision, hash: %d, cursors: %d", c.hash, len(cursors))
	}
	return c
}

// Add a new cursor to the collection of cursors by cursor hash and it's index in the collection
func (cm *cursorMap) cursorIndex(c *cursor) int {
	cursors, exists := cm.cursors[c.hash]
	if !exists {
		return -1
	}
	for i, c2 := range cursors {
		if c2 == c {
			return i
		}
	}
	return -1
}

// Add a new cursor to the collection of cursors by cursor hash and it's index in the collection
func (cm *cursorMap) deleteCursor(c *cursor) error {
	cursors, exists := cm.cursors[c.hash]
	if !exists {
		return fmt.Errorf("RoundRobin: cursor not found")
	}
	if len(cursors) == 1 {
		delete(cm.cursors, c.hash)
	}
	index := cm.cursorIndex(c)
	if index == -1 {
		return fmt.Errorf("Cursor not found")
	}
	cm.cursors[c.hash] = append(cursors[:index], cursors[index+1:]...)
	return nil
}

// Computes the hash of the cursor by computing the hash of every enpdoint supplied
func computeHash(endpoints []loadbalance.Endpoint) uint32 {
	h := fnv.New32()
	for _, endpoint := range endpoints {
		h.Write([]byte(endpoint.Id()))
	}
	return h.Sum32()
}

// This function checks if there a
func (cm *cursorMap) deleteExpiredCursors(now int) {
	glog.Infof("RoundRobin GC: start: %d cursors, expiry times: %d", len(cm.cursors), cm.expiryTimes.Len())
	for {
		if cm.expiryTimes.Len() == 0 {
			break
		}
		item := cm.expiryTimes.PeekEl()
		if item.Priority > now {
			glog.Infof("RoundRobin GC: Nothing to expire, earliest expiry is: Cursor(%v, lastAccess=%d), now is %d", item.Value, item.Priority, now)
			break
		} else {
			glog.Infof("RoundRobin GC: Cursor(%v, lastAccess=%d) has expired (now=%d), deleting", item.Value, item.Priority, now)
			el := cm.expiryTimes.PopEl()
			cursor := (el.Value).(*cursor)
			cm.deleteCursor(cursor)
		}
	}
	glog.Infof("RoundRobin GC end: %d cursors, expiry times: %d", len(cm.cursors), cm.expiryTimes.Len())
}

// Cursor represents the current position in the given endpoints sequence
type cursor struct {
	// position in the upstreams
	index       int
	hash        uint32
	item        *minheap.Element
	endpointIds []string
}

func newCursor(endpoints []loadbalance.Endpoint) *cursor {
	endpointIds := make([]string, len(endpoints))
	for i, endpoint := range endpoints {
		endpointIds[i] = endpoint.Id()
	}
	return &cursor{
		index:       0,
		hash:        computeHash(endpoints),
		endpointIds: endpointIds,
	}
}

func (c *cursor) sameEndpoints(endpoints []loadbalance.Endpoint) bool {
	if len(c.endpointIds) != len(endpoints) {
		return false
	}
	for i, _ := range endpoints {
		if c.endpointIds[i] != endpoints[i].Id() {
			return false
		}
	}
	return true
}

func (c *cursor) next(endpoints []loadbalance.Endpoint) (loadbalance.Endpoint, error) {
	for i := 0; i < len(endpoints); i++ {
		endpoint := endpoints[c.index]
		c.index = (c.index + 1) % len(endpoints)
		if endpoint.IsActive() {
			return endpoint, nil
		} else {
			glog.Infof("Skipping inactive endpoint: %s", endpoint.Id())
		}
	}
	// That means that we did full circle and found nothing
	return nil, fmt.Errorf("No available endpoints!")
}
