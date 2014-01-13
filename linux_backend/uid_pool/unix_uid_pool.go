package uid_pool

import (
	"fmt"
	"sync"
)

type UnixUIDPool struct {
	start uint32
	size  uint32

	pool      []uint32
	poolMutex *sync.Mutex
}

type PoolExhaustedError struct{}

func (e PoolExhaustedError) Error() string {
	return "UID pool is exhausted"
}

type UIDTakenError struct {
	UID uint32
}

func (e UIDTakenError) Error() string {
	return fmt.Sprintf("uid already acquired: %d", e.UID)
}

func New(start, size uint32) *UnixUIDPool {
	pool := []uint32{}

	for i := start; i < start+size; i++ {
		pool = append(pool, i)
	}

	return &UnixUIDPool{
		start: start,
		size:  size,

		pool:      pool,
		poolMutex: new(sync.Mutex),
	}
}

func (p *UnixUIDPool) Acquire() (uint32, error) {
	p.poolMutex.Lock()
	defer p.poolMutex.Unlock()

	if len(p.pool) == 0 {
		return 0, PoolExhaustedError{}
	}

	uid := p.pool[0]

	p.pool = p.pool[1:]

	return uid, nil
}

func (p *UnixUIDPool) Remove(uid uint32) error {
	idx := 0
	found := false

	p.poolMutex.Lock()
	defer p.poolMutex.Unlock()

	for i, existingUID := range p.pool {
		if existingUID == uid {
			idx = i
			found = true
			break
		}
	}

	if !found {
		return UIDTakenError{uid}
	}

	p.pool = append(p.pool[:idx], p.pool[idx+1:]...)

	return nil
}

func (p *UnixUIDPool) Release(uid uint32) {
	if uid < p.start || uid >= p.start+p.size {
		return
	}

	p.poolMutex.Lock()
	defer p.poolMutex.Unlock()

	p.pool = append(p.pool, uid)
}
