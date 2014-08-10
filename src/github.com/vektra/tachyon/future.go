package tachyon

import (
	"sync"
	"time"
)

type Future struct {
	Task    *Task
	Start   time.Time
	Runtime time.Duration

	result *Result
	err    error
	wg     sync.WaitGroup
}

func NewFuture(start time.Time, task *Task, f func() (*Result, error)) *Future {
	fut := &Future{Start: start, Task: task}

	fut.wg.Add(1)

	go func() {
		r, e := f()
		fut.result = r
		fut.err = e
		fut.Runtime = time.Since(fut.Start)
		fut.wg.Done()
	}()

	return fut
}

func (f *Future) Wait() {
	f.wg.Wait()
}

func (f *Future) Value() (*Result, error) {
	f.Wait()
	return f.result, f.err
}

func (f *Future) Read() interface{} {
	f.Wait()
	return f.result
}

type Futures map[string]*Future

type FutureScope struct {
	Scope
	futures Futures
}

func NewFutureScope(parent Scope) *FutureScope {
	return &FutureScope{
		Scope:   parent,
		futures: Futures{},
	}
}

func (fs *FutureScope) Get(key string) (Value, bool) {
	if v, ok := fs.futures[key]; ok {
		return v, ok
	}

	return fs.Scope.Get(key)
}

func (fs *FutureScope) AddFuture(key string, f *Future) {
	fs.futures[key] = f
}

func (fs *FutureScope) Wait() {
	for _, f := range fs.futures {
		f.Wait()
	}
}

func (fs *FutureScope) Results() []RunResult {
	var results []RunResult

	for _, f := range fs.futures {
		f.Wait()

		results = append(results, RunResult{f.Task, f.result, f.Runtime})
	}

	return results
}
