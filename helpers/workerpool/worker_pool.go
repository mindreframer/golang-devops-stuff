package workerpool

import (
	"sync"
)

type WorkerPool struct {
	workerChannels []chan func()
	index          int
	indexLock      *sync.Mutex
	stopped        bool
}

func NewWorkerPool(poolSize int) (pool *WorkerPool) {
	pool = &WorkerPool{
		workerChannels: make([]chan func(), poolSize),
		indexLock:      &sync.Mutex{},
	}

	for i := range pool.workerChannels {
		pool.workerChannels[i] = make(chan func(), 0)
		go pool.startWorker(pool.workerChannels[i])
	}

	return
}

func (pool *WorkerPool) ScheduleWork(work func()) {
	if pool.stopped {
		return
	}

	go func() {
		pool.indexLock.Lock()
		index := pool.index
		pool.index = (pool.index + 1) % len(pool.workerChannels)
		pool.indexLock.Unlock()

		pool.workerChannels[index] <- work
	}()
}

func (pool *WorkerPool) StopWorkers() {
	pool.stopped = true
	for _, workerChannel := range pool.workerChannels {
		close(workerChannel)
	}
}

func (pool *WorkerPool) startWorker(workerChannel chan func()) {
	for {
		f, ok := <-workerChannel
		if ok {
			f()
		} else {
			return
		}
	}
}
