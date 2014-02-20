package tachyon

import (
	"fmt"
)

type AsyncAction struct {
	Task   *Task
	Error  error
	status chan *AsyncAction
}

func (a *AsyncAction) Init(pe *PlayEnv) {
	pe.wait.Add(1)
	a.status = pe.AsyncChannel()
}

func (a *AsyncAction) Finish(err error) {
	a.Error = err
	a.status <- a
}

func (pe *PlayEnv) handleAsync() {
	for {
		act := <-pe.async

		if act.Error == nil {
			fmt.Printf("- %s (async success)\n", act.Task.Name())

			for _, x := range act.Task.Notify() {
				pe.AddNotify(x)
			}
		} else {
			fmt.Printf("- %s (async error:%s)\n", act.Task.Name(), act.Error)
		}

		pe.wait.Done()
	}
}
