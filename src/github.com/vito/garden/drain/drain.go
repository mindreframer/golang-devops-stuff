package drain

import (
	"sync"
	"sync/atomic"
)

type Drain struct {
	group *sync.Cond
	count int64
}

func New() *Drain {
	return &Drain{
		group: sync.NewCond(&sync.Mutex{}),
	}
}

func (d *Drain) Incr() {
	atomic.AddInt64(&d.count, 1)
}

func (d *Drain) Decr() {
	cur := atomic.AddInt64(&d.count, -1)
	if cur < 0 {
		panic(".Decr more than .Incr")
	}

	if cur == 0 {
		d.group.L.Lock()
		defer d.group.L.Unlock()

		d.group.Broadcast()
	}
}

func (d *Drain) Wait() {
	d.group.L.Lock()

	for !d.isClear() {
		d.group.Wait()
	}

	d.group.L.Unlock()
}

func (d *Drain) isClear() bool {
	return atomic.LoadInt64(&d.count) == 0
}
