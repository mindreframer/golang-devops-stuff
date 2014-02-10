package fake_container_pool

import (
	"fmt"
	"io"

	"github.com/pivotal-cf-experimental/garden/backend"
	"github.com/pivotal-cf-experimental/garden/backend/fake_backend"
	"github.com/pivotal-cf-experimental/garden/linux_backend"
)

type FakeContainerPool struct {
	DidSetup bool

	Pruned         bool
	PruneError     error
	KeptContainers map[string]bool

	CreateError  error
	RestoreError error
	DestroyError error

	ContainerSetup func(*fake_backend.FakeContainer)

	CreatedContainers   []linux_backend.Container
	DestroyedContainers []linux_backend.Container
	RestoredSnapshots   []io.Reader
}

func New() *FakeContainerPool {
	return &FakeContainerPool{}
}

func (p *FakeContainerPool) Setup() error {
	p.DidSetup = true

	return nil
}

func (p *FakeContainerPool) Prune(keep map[string]bool) error {
	if p.PruneError != nil {
		return p.PruneError
	}

	p.Pruned = true
	p.KeptContainers = keep

	return nil
}

func (p *FakeContainerPool) Create(spec backend.ContainerSpec) (linux_backend.Container, error) {
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

func (p *FakeContainerPool) Restore(snapshot io.Reader) (linux_backend.Container, error) {
	if p.RestoreError != nil {
		return nil, p.RestoreError
	}

	var handle string

	_, err := fmt.Fscanf(snapshot, "%s", &handle)
	if err != nil && err != io.EOF {
		return nil, err
	}

	container := fake_backend.NewFakeContainer(
		backend.ContainerSpec{
			Handle: handle,
		},
	)

	p.RestoredSnapshots = append(p.RestoredSnapshots, snapshot)

	return container, nil
}

func (p *FakeContainerPool) Destroy(container linux_backend.Container) error {
	if p.DestroyError != nil {
		return p.DestroyError
	}

	p.DestroyedContainers = append(p.DestroyedContainers, container)

	return nil
}
