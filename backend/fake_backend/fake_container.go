package fake_backend

import (
	"github.com/vito/garden/backend"
)

type FakeContainer struct {
	Spec backend.ContainerSpec

	StartError error
	Started    bool

	StopError    error
	Stopped      []StopSpec
	StopCallback func()

	CopyInError error
	CopiedIn    [][]string

	CopyOutError error
	CopiedOut    [][]string

	SpawnError   error
	SpawnedJobID uint32
	Spawned      []backend.JobSpec

	LinkError       error
	LinkedJobResult backend.JobResult
	Linked          []uint32

	StreamError       error
	StreamedJobChunks []backend.JobStream
	Streamed          []uint32

	LimitBandwidthError error
	LimitedBandwidth    backend.BandwidthLimits

	LimitMemoryError error
	LimitedMemory    backend.MemoryLimits

	LimitDiskError    error
	LimitedDisk       backend.DiskLimits
	LimitedDiskResult backend.DiskLimits

	NetInError error
	MappedIn   [][]uint32

	NetOutError  error
	PermittedOut []NetOutSpec

	InfoError    error
	ReportedInfo backend.ContainerInfo
}

type NetOutSpec struct {
	Network string
	Port    uint32
}

type StopSpec struct {
	Killed bool
}

func NewFakeContainer(spec backend.ContainerSpec) *FakeContainer {
	return &FakeContainer{Spec: spec}
}

func (c *FakeContainer) ID() string {
	return c.Spec.Handle
}

func (c *FakeContainer) Handle() string {
	return c.Spec.Handle
}

func (c *FakeContainer) Start() error {
	if c.StartError != nil {
		return c.StartError
	}

	c.Started = true

	return nil
}

func (c *FakeContainer) Stop(kill bool) error {
	if c.StopError != nil {
		return c.StopError
	}

	if c.StopCallback != nil {
		c.StopCallback()
	}

	c.Stopped = append(c.Stopped, StopSpec{kill})

	return nil
}

func (c *FakeContainer) Info() (backend.ContainerInfo, error) {
	if c.InfoError != nil {
		return backend.ContainerInfo{}, c.InfoError
	}

	return c.ReportedInfo, nil
}

func (c *FakeContainer) CopyIn(src, dst string) error {
	if c.CopyInError != nil {
		return c.CopyInError
	}

	c.CopiedIn = append(c.CopiedIn, []string{src, dst})

	return nil
}

func (c *FakeContainer) CopyOut(src, dst, owner string) error {
	if c.CopyOutError != nil {
		return c.CopyOutError
	}

	c.CopiedOut = append(c.CopiedOut, []string{src, dst, owner})

	return nil
}

func (c *FakeContainer) LimitBandwidth(limits backend.BandwidthLimits) (backend.BandwidthLimits, error) {
	if c.LimitBandwidthError != nil {
		return backend.BandwidthLimits{}, c.LimitBandwidthError
	}

	c.LimitedBandwidth = limits

	return limits, nil
}

func (c *FakeContainer) LimitDisk(limits backend.DiskLimits) (backend.DiskLimits, error) {
	if c.LimitDiskError != nil {
		return backend.DiskLimits{}, c.LimitDiskError
	}

	c.LimitedDisk = limits

	if c.LimitedDiskResult != limits {
		return c.LimitedDiskResult, nil
	}

	return limits, nil
}

func (c *FakeContainer) LimitMemory(limits backend.MemoryLimits) (backend.MemoryLimits, error) {
	if c.LimitMemoryError != nil {
		return backend.MemoryLimits{}, c.LimitMemoryError
	}

	c.LimitedMemory = limits

	return limits, nil
}

func (c *FakeContainer) Spawn(spec backend.JobSpec) (uint32, error) {
	if c.SpawnError != nil {
		return 0, c.SpawnError
	}

	c.Spawned = append(c.Spawned, spec)

	return c.SpawnedJobID, nil
}

func (c *FakeContainer) Stream(jobID uint32) (<-chan backend.JobStream, error) {
	if c.StreamError != nil {
		return nil, c.StreamError
	}

	c.Streamed = append(c.Streamed, jobID)

	stream := make(chan backend.JobStream, len(c.StreamedJobChunks))

	for _, chunk := range c.StreamedJobChunks {
		stream <- chunk
	}

	close(stream)

	return stream, nil
}

func (c *FakeContainer) Link(jobID uint32) (backend.JobResult, error) {
	if c.LinkError != nil {
		return backend.JobResult{}, c.LinkError
	}

	c.Linked = append(c.Linked, jobID)

	return c.LinkedJobResult, nil
}

func (c *FakeContainer) NetIn(hostPort uint32, containerPort uint32) (uint32, uint32, error) {
	if c.NetInError != nil {
		return 0, 0, c.NetInError
	}

	c.MappedIn = append(c.MappedIn, []uint32{hostPort, containerPort})

	return hostPort, containerPort, nil
}

func (c *FakeContainer) NetOut(network string, port uint32) error {
	if c.NetOutError != nil {
		return c.NetOutError
	}

	c.PermittedOut = append(c.PermittedOut, NetOutSpec{network, port})

	return nil
}
