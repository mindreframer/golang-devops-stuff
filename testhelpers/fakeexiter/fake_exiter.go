package fakeexiter

type FakeExiter struct {
	DidExit    bool
	ExitStatus int
}

func New() *FakeExiter {
	return &FakeExiter{}
}

func (e *FakeExiter) Exit(status int) {
	e.DidExit = true
	e.ExitStatus = status
}
