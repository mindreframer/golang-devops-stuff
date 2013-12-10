package backend

import (
	"github.com/mailgun/vulcan/timeutils"
	"time"
)

// Stats backend, used to memorize counters for given keys to
// to request stats about upstreams and tokens
type Backend interface {
	// Used to retreive time for current stats.
	// Creates mostly for test reasons so we can override time in tests
	timeutils.TimeProvider

	// Get count of the given key in the time period
	GetCount(key string, period time.Duration) (int64, error)

	// Updates hitcount of the given key, with the given increment
	UpdateCount(key string, period time.Duration, increment int64) error
}
