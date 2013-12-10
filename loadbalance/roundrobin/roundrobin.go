/* Implements round robin load balancing algorithm.

* As long as vulcan does not have static endpoints configurations most of the time,
it keeps track of the endpoints that were used recently and keeps cursor for these endpoints for a while.

* Unused cursors are being expired and removed from the map if they have not been used
for 60 seconds.

* If the load balancer can find matching cursor for the given endpoints, algo simply advances to the next one,
taking into consideration endpoint availability.
*/
package roundrobin

import (
	"fmt"
	"github.com/mailgun/vulcan/loadbalance"
	"github.com/mailgun/vulcan/timeutils"
	"sync"
)

type RoundRobin struct {
	// time provider (mostly for testing as we need to override time
	timeProvider timeutils.TimeProvider

	// keep in mind that load balancer used by different endpoints
	mutex *sync.Mutex

	// collection of cursors
	cursors *cursorMap
}

func NewRoundRobin(timeProvider timeutils.TimeProvider) *RoundRobin {

	return &RoundRobin{
		timeProvider: timeProvider,
		cursors:      newCursorMap(),
		mutex:        &sync.Mutex{},
	}
}

const ExpirySeconds = 60

func (r *RoundRobin) NextEndpoint(endpoints []loadbalance.Endpoint) (loadbalance.Endpoint, error) {
	if len(endpoints) == 0 {
		return nil, fmt.Errorf("Need some endpoints")
	}
	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.cursors.deleteExpiredCursors(int(r.timeProvider.UtcNow().Unix()))

	// Get existing cursor or create new cursor
	expirySeconds := int(r.timeProvider.UtcNow().Unix()) + ExpirySeconds
	c := r.cursors.upsertCursor(endpoints, expirySeconds)

	// Return the next endpoint referred by this cursor
	return c.next(endpoints)
}
