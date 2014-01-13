package port_pool

import (
	"fmt"
	"sync"
)

type PortPool struct {
	start uint32
	size  uint32

	pool      []uint32
	poolMutex sync.Mutex
}

type PoolExhaustedError struct{}

func (e PoolExhaustedError) Error() string {
	return "port pool is exhausted"
}

type PortTakenError struct {
	Port uint32
}

func (e PortTakenError) Error() string {
	return fmt.Sprintf("port already acquired: %d", e.Port)
}

func New(start, size uint32) *PortPool {
	pool := []uint32{}

	for i := start; i < start+size; i++ {
		pool = append(pool, i)
	}

	return &PortPool{
		start: start,
		size:  size,

		pool: pool,
	}
}

func (p *PortPool) Acquire() (uint32, error) {
	p.poolMutex.Lock()
	defer p.poolMutex.Unlock()

	if len(p.pool) == 0 {
		return 0, PoolExhaustedError{}
	}

	port := p.pool[0]

	p.pool = p.pool[1:]

	return port, nil
}

func (p *PortPool) Remove(port uint32) error {
	idx := 0
	found := false

	p.poolMutex.Lock()
	defer p.poolMutex.Unlock()

	for i, existingPort := range p.pool {
		if existingPort == port {
			idx = i
			found = true
			break
		}
	}

	if !found {
		return PortTakenError{port}
	}

	p.pool = append(p.pool[:idx], p.pool[idx+1:]...)

	return nil
}

func (p *PortPool) Release(port uint32) {
	if port < p.start || port >= p.start+p.size {
		return
	}

	p.poolMutex.Lock()
	defer p.poolMutex.Unlock()

	for _, existingPort := range p.pool {
		if existingPort == port {
			return
		}
	}

	p.pool = append(p.pool, port)
}
