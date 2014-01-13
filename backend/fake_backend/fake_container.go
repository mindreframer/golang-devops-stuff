package fake_backend

import (
	"io"
	"sync"
	"time"

	"github.com/vito/garden/backend"
)

type FakeContainer struct {
	Spec backend.ContainerSpec

	StartError error
	Started    bool

	StopError    error
	stopped      []StopSpec
	stopMutex    *sync.RWMutex
	StopCallback func()

	CleanedUp bool

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
	StreamDelay       time.Duration

	DidLimitBandwidth   bool
	LimitBandwidthError error
	LimitedBandwidth    backend.BandwidthLimits

	CurrentBandwidthLimitsResult backend.BandwidthLimits
	CurrentBandwidthLimitsError  error

	DidLimitMemory   bool
	LimitMemoryError error
	LimitedMemory    backend.MemoryLimits

	CurrentMemoryLimitsResult backend.MemoryLimits
	CurrentMemoryLimitsError  error

	DidLimitDisk   bool
	LimitDiskError error
	LimitedDisk    backend.DiskLimits

	CurrentDiskLimitsResult backend.DiskLimits
	CurrentDiskLimitsError  error

	DidLimitCPU   bool
	LimitCPUError error
	LimitedCPU    backend.CPULimits

	CurrentCPULimitsResult backend.CPULimits
	CurrentCPULimitsError  error

	NetInError error
	MappedIn   [][]uint32

	NetOutError  error
	PermittedOut []NetOutSpec

	InfoError    error
	ReportedInfo backend.ContainerInfo

	SnapshotError  error
	SavedSnapshots []io.Writer
	snapshotMutex  *sync.RWMutex
}

type NetOutSpec struct {
	Network string
	Port    uint32
}

type StopSpec struct {
	Killed bool
}

func NewFakeContainer(spec backend.ContainerSpec) *FakeContainer {
	return &FakeContainer{
		Spec: spec,

		stopMutex:     new(sync.RWMutex),
		snapshotMutex: new(sync.RWMutex),
	}
}

func (c *FakeContainer) ID() string {
	return c.Spec.Handle
}

func (c *FakeContainer) Handle() string {
	return c.Spec.Handle
}

func (c *FakeContainer) GraceTime() time.Duration {
	return c.Spec.GraceTime
}

func (c *FakeContainer) Snapshot(snapshot io.Writer) error {
	if c.SnapshotError != nil {
		return c.SnapshotError
	}

	c.snapshotMutex.Lock()
	defer c.snapshotMutex.Unlock()

	c.SavedSnapshots = append(c.SavedSnapshots, snapshot)

	return nil
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

	// stops can happen asynchronously in tests (i.e. StopRequest with
	// Background: true), so we need a mutex here
	c.stopMutex.Lock()
	defer c.stopMutex.Unlock()

	c.stopped = append(c.stopped, StopSpec{kill})

	return nil
}

func (c *FakeContainer) Stopped() []StopSpec {
	c.stopMutex.RLock()
	defer c.stopMutex.RUnlock()

	return c.stopped
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

func (c *FakeContainer) LimitBandwidth(limits backend.BandwidthLimits) error {
	c.DidLimitBandwidth = true

	if c.LimitBandwidthError != nil {
		return c.LimitBandwidthError
	}

	c.LimitedBandwidth = limits

	return nil
}

func (c *FakeContainer) CurrentBandwidthLimits() (backend.BandwidthLimits, error) {
	if c.CurrentBandwidthLimitsError != nil {
		return backend.BandwidthLimits{}, c.CurrentBandwidthLimitsError
	}

	return c.CurrentBandwidthLimitsResult, nil
}

func (c *FakeContainer) LimitDisk(limits backend.DiskLimits) error {
	c.DidLimitDisk = true

	if c.LimitDiskError != nil {
		return c.LimitDiskError
	}

	c.LimitedDisk = limits

	return nil
}

func (c *FakeContainer) CurrentDiskLimits() (backend.DiskLimits, error) {
	if c.CurrentDiskLimitsError != nil {
		return backend.DiskLimits{}, c.CurrentDiskLimitsError
	}

	return c.CurrentDiskLimitsResult, nil
}

func (c *FakeContainer) LimitMemory(limits backend.MemoryLimits) error {
	c.DidLimitMemory = true

	if c.LimitMemoryError != nil {
		return c.LimitMemoryError
	}

	c.LimitedMemory = limits

	return nil
}

func (c *FakeContainer) CurrentMemoryLimits() (backend.MemoryLimits, error) {
	if c.CurrentMemoryLimitsError != nil {
		return backend.MemoryLimits{}, c.CurrentMemoryLimitsError
	}

	return c.CurrentMemoryLimitsResult, nil
}

func (c *FakeContainer) LimitCPU(limits backend.CPULimits) error {
	c.DidLimitCPU = true

	if c.LimitCPUError != nil {
		return c.LimitCPUError
	}

	c.LimitedCPU = limits

	return nil
}

func (c *FakeContainer) CurrentCPULimits() (backend.CPULimits, error) {
	if c.CurrentCPULimitsError != nil {
		return backend.CPULimits{}, c.CurrentCPULimitsError
	}

	return c.CurrentCPULimitsResult, nil
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
		time.Sleep(c.StreamDelay)
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

func (c *FakeContainer) Cleanup() {
	c.CleanedUp = true
}
