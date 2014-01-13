package fakelocker

type FakeLocker struct {
	GetAndMaintainLockError error
	GotAndMaintainedLock    bool

	ReleasedLock bool
}

func New() *FakeLocker {
	return &FakeLocker{}
}

func (l *FakeLocker) GetAndMaintainLock() error {
	if l.GetAndMaintainLockError != nil {
		return l.GetAndMaintainLockError
	}

	l.GotAndMaintainedLock = true

	return nil
}

func (l *FakeLocker) ReleaseLock() {
	l.ReleasedLock = true
}
