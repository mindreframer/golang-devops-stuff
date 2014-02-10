package fake_backend

import (
	"time"

	"github.com/pivotal-cf-experimental/garden/backend"
)

type SlowFakeBackend struct {
	FakeBackend

	delay time.Duration
}

func NewSlow(delay time.Duration) *SlowFakeBackend {
	return &SlowFakeBackend{
		FakeBackend: *New(),

		delay: delay,
	}
}

func (b *SlowFakeBackend) Create(spec backend.ContainerSpec) (backend.Container, error) {
	time.Sleep(b.delay)

	return b.FakeBackend.Create(spec)
}

func (b *SlowFakeBackend) Lookup(handle string) (backend.Container, error) {
	time.Sleep(b.delay)

	return b.FakeBackend.Lookup(handle)
}
