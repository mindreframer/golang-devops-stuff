package linux_backend

import (
	"time"

	"github.com/vito/garden/backend"
	"github.com/vito/garden/linux_backend/network"
)

type ContainerSnapshot struct {
	ID     string
	Handle string

	GraceTime time.Duration

	State  string
	Events []string

	Limits LimitsSnapshot

	Resources ResourcesSnapshot

	Jobs []JobSnapshot

	NetIns  []NetInSpec
	NetOuts []NetOutSpec
}

type LimitsSnapshot struct {
	Memory    *backend.MemoryLimits
	Disk      *backend.DiskLimits
	Bandwidth *backend.BandwidthLimits
	CPU       *backend.CPULimits
}

type ResourcesSnapshot struct {
	UID     uint32
	Network *network.Network
	Ports   []uint32
}

type JobSnapshot struct {
	ID            uint32
	DiscardOutput bool
}
