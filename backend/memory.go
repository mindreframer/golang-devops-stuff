/*
Memory backend, used mostly for testing, but may be extended to become
more useful in the future. In this case it'll need garbage collection.
*/
package backend

import (
	"github.com/mailgun/vulcan/timeutils"
	"time"
)

type MemoryBackend struct {
	Hits         map[string]int64
	TimeProvider timeutils.TimeProvider
}

func NewMemoryBackend(timeProvider timeutils.TimeProvider) (*MemoryBackend, error) {
	return &MemoryBackend{
		Hits:         map[string]int64{},
		TimeProvider: timeProvider,
	}, nil
}

func (b *MemoryBackend) GetCount(key string, period time.Duration) (int64, error) {
	return b.Hits[timeutils.GetHit(b.UtcNow(), key, period)], nil
}

func (b *MemoryBackend) UpdateCount(key string, period time.Duration, increment int64) error {
	b.Hits[timeutils.GetHit(b.UtcNow(), key, period)] += increment
	return nil
}

func (b *MemoryBackend) UtcNow() time.Time {
	return b.TimeProvider.UtcNow()
}
