package linux_backend

import (
	"sync"

	"github.com/vito/garden/backend"
)

type ContainerPool interface {
	Setup() error
	Create(backend.ContainerSpec) (backend.Container, error)
	Destroy(backend.Container) error
}

type LinuxBackend struct {
	containerPool ContainerPool

	containers map[string]backend.Container

	sync.RWMutex
}

type UnknownHandleError struct {
	Handle string
}

func (e UnknownHandleError) Error() string {
	return "unknown handle: " + e.Handle
}

func New(containerPool ContainerPool) *LinuxBackend {
	return &LinuxBackend{
		containerPool: containerPool,

		containers: make(map[string]backend.Container),
	}
}

func (b *LinuxBackend) Setup() error {
	return b.containerPool.Setup()
}

func (b *LinuxBackend) Create(spec backend.ContainerSpec) (backend.Container, error) {
	container, err := b.containerPool.Create(spec)
	if err != nil {
		return nil, err
	}

	err = container.Start()
	if err != nil {
		return nil, err
	}

	b.Lock()

	b.containers[container.Handle()] = container

	b.Unlock()

	return container, nil
}

func (b *LinuxBackend) Destroy(handle string) error {
	container, found := b.containers[handle]
	if !found {
		return UnknownHandleError{handle}
	}

	err := b.containerPool.Destroy(container)
	if err != nil {
		return err
	}

	b.Lock()

	delete(b.containers, container.Handle())

	b.Unlock()

	return nil
}

func (b *LinuxBackend) Containers() (containers []backend.Container, err error) {
	b.RLock()
	defer b.RUnlock()

	for _, container := range b.containers {
		containers = append(containers, container)
	}

	return containers, nil
}

func (b *LinuxBackend) Lookup(handle string) (backend.Container, error) {
	b.RLock()
	defer b.RUnlock()

	container, found := b.containers[handle]
	if !found {
		return nil, UnknownHandleError{handle}
	}

	return container, nil
}
