package fake_backend

import (
	"io"
	"sync"

	"github.com/vito/garden/backend"
)

type FakeBackend struct {
	Started    bool
	StartError error

	Stopped bool

	CreateResult    *FakeContainer
	CreateError     error
	RestoreError    error
	DestroyError    error
	ContainersError error

	CreatedContainers   map[string]*FakeContainer
	DestroyedContainers []string
	RestoredContainers  []io.Reader

	sync.RWMutex
}

type UnknownHandleError struct {
	Handle string
}

func (e UnknownHandleError) Error() string {
	return "unknown handle: " + e.Handle
}

func New() *FakeBackend {
	return &FakeBackend{
		CreatedContainers: make(map[string]*FakeContainer),
	}
}

func (b *FakeBackend) Setup() error {
	return nil
}

func (b *FakeBackend) Start() error {
	if b.StartError != nil {
		return b.StartError
	}

	b.Started = true

	return nil
}

func (b *FakeBackend) Stop() {
	b.Stopped = true
}

func (b *FakeBackend) Create(spec backend.ContainerSpec) (backend.Container, error) {
	if b.CreateError != nil {
		return nil, b.CreateError
	}

	var container *FakeContainer

	if b.CreateResult != nil {
		container = b.CreateResult
	} else {
		container = NewFakeContainer(spec)
	}

	b.Lock()
	defer b.Unlock()

	b.CreatedContainers[container.Handle()] = container

	return container, nil
}

func (b *FakeBackend) Restore(snapshot io.Reader) (backend.Container, error) {
	if b.RestoreError != nil {
		return nil, b.RestoreError
	}

	b.Lock()
	defer b.Unlock()

	b.RestoredContainers = append(b.RestoredContainers, snapshot)

	return NewFakeContainer(backend.ContainerSpec{}), nil
}

func (b *FakeBackend) Destroy(handle string) error {
	if b.DestroyError != nil {
		return b.DestroyError
	}

	b.Lock()
	defer b.Unlock()

	delete(b.CreatedContainers, handle)

	b.DestroyedContainers = append(b.DestroyedContainers, handle)

	return nil
}

func (b *FakeBackend) Containers() (containers []backend.Container, err error) {
	if b.ContainersError != nil {
		err = b.ContainersError
		return
	}

	b.RLock()
	defer b.RUnlock()

	for _, c := range b.CreatedContainers {
		containers = append(containers, c)
	}

	return
}

func (b *FakeBackend) Lookup(handle string) (backend.Container, error) {
	b.RLock()
	defer b.RUnlock()

	container, found := b.CreatedContainers[handle]
	if !found {
		return nil, UnknownHandleError{handle}
	}

	return container, nil
}
