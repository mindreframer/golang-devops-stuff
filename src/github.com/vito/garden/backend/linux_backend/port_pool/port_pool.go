package port_pool

import (
	"sync"
)

type PortPool struct {
	start uint32
	size  uint32

	pool []uint32

	sync.Mutex
}

type PoolExhaustedError struct{}

func (e PoolExhaustedError) Error() string {
	return "Port pool is exhausted"
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
	p.Lock()
	defer p.Unlock()

	if len(p.pool) == 0 {
		return 0, PoolExhaustedError{}
	}

	port := p.pool[0]

	p.pool = p.pool[1:]

	return port, nil
}

func (p *PortPool) Release(port uint32) {
	if port < p.start || port >= p.start+p.size {
		return
	}

	p.Lock()
	defer p.Unlock()

	p.pool = append(p.pool, port)
}
