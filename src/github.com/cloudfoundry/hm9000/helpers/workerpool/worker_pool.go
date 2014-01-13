package workerpool

import (
	"sync"
	"time"
)

type WorkerPool struct {
	workerChannels []chan func()
	indexProvider  chan chan int
	timeLock       *sync.Mutex
	stopped        bool

	timeSpentWorking     time.Duration
	usageSampleStartTime time.Time
}

func NewWorkerPool(poolSize int) (pool *WorkerPool) {
	pool = &WorkerPool{
		workerChannels: make([]chan func(), poolSize),
		indexProvider:  make(chan chan int, 0),
		timeLock:       &sync.Mutex{},
	}

	pool.resetUsageTracking()
	go pool.mux()

	for i := range pool.workerChannels {
		pool.workerChannels[i] = make(chan func(), 0)
		go pool.startWorker(pool.workerChannels[i])
	}

	return
}

func (pool *WorkerPool) mux() {
	index := 0
	for {
		select {
		case c := <-pool.indexProvider:
			go func(index int) {
				c <- index
			}(index)
			index = (index + 1) % len(pool.workerChannels)
		}
	}
}

func (pool *WorkerPool) getNextIndex() int {
	c := make(chan int, 1)
	pool.indexProvider <- c
	return <-c
}

func (pool *WorkerPool) ScheduleWork(work func()) {
	if pool.stopped {
		return
	}

	go func() {
		index := pool.getNextIndex()
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
			tWork := time.Now()
			f()
			dtWork := time.Since(tWork)

			pool.timeLock.Lock()
			pool.timeSpentWorking += dtWork
			pool.timeLock.Unlock()
		} else {
			return
		}
	}
}

func (pool *WorkerPool) StartTrackingUsage() {
	pool.resetUsageTracking()
}

func (pool *WorkerPool) MeasureUsage() (usage float64, measurementDuration time.Duration) {
	pool.timeLock.Lock()
	timeSpentWorking := pool.timeSpentWorking
	measurementDuration = time.Since(pool.usageSampleStartTime)
	pool.timeLock.Unlock()

	usage = timeSpentWorking.Seconds() / (measurementDuration.Seconds() * float64(len(pool.workerChannels)))

	pool.resetUsageTracking()
	return usage, measurementDuration
}

func (pool *WorkerPool) resetUsageTracking() {
	pool.timeLock.Lock()
	pool.usageSampleStartTime = time.Now()
	pool.timeSpentWorking = 0
	pool.timeLock.Unlock()
}
