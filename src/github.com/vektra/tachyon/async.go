package tachyon

type AsyncAction struct {
	Task   *Task
	Error  error
	Result *Result
	status chan *AsyncAction
}

func (a *AsyncAction) Init(r *Runner) {
	r.wait.Add(1)
	a.status = r.AsyncChannel()
}

func (a *AsyncAction) Finish(res *Result, err error) {
	a.Error = err
	a.Result = res
	a.status <- a
}

func (r *Runner) handleAsync() {
	for {
		act := <-r.async

		r.env.report.FinishAsyncTask(act)

		if act.Error == nil {
			for _, x := range act.Task.Notify() {
				r.AddNotify(x)
			}
		}

		r.wait.Done()
	}
}
