package backend

import (
	"fmt"
	"time"
)

/*
All operation on this backend always fail
*/
type FailingBackend struct {
}

func (b *FailingBackend) GetCount(key string, period time.Duration) (int64, error) {
	return -1, fmt.Errorf("Something went wrong")
}

func (b *FailingBackend) UpdateCount(key string, period time.Duration, increment int64) error {
	return fmt.Errorf("Something went wrong")
}

func (b *FailingBackend) UtcNow() time.Time {
	return time.Now().UTC()
}
