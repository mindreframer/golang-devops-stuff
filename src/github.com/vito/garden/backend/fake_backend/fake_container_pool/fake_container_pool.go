package fake_container_pool

import (
	"github.com/vito/garden/backend"
	"github.com/vito/garden/backend/fake_backend"
)

type FakeContainerPool struct {
	DidSetup bool

	CreateError  error
	DestroyError error

	ContainerSetup func(*fake_backend.FakeContainer)

	CreatedContainers   []backend.Container
	DestroyedContainers []backend.Container
}

func New() *FakeContainerPool {
	return &FakeContainerPool{}
}

func (p *FakeContainerPool) Setup() error {
	p.DidSetup = true

	return nil
}

func (p *FakeContainerPool) Create(spec backend.ContainerSpec) (backend.Container, error) {
	if p.CreateError != nil {
		return nil, p.CreateError
	}

	container := fake_backend.NewFakeContainer(spec)

	if p.ContainerSetup != nil {
		p.ContainerSetup(container)
	}

	p.CreatedContainers = append(p.CreatedContainers, container)

	return container, nil
}

func (p *FakeContainerPool) Destroy(container backend.Container) error {
	if p.DestroyError != nil {
		return p.DestroyError
	}

	p.DestroyedContainers = append(p.DestroyedContainers, container)

	return nil
}
