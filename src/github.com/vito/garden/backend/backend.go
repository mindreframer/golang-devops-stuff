package backend

import (
	"time"
)

type Backend interface {
	Setup() error
	Create(ContainerSpec) (Container, error)
	Destroy(handle string) error
	Containers() ([]Container, error)
	Lookup(handle string) (Container, error)
}

type ContainerSpec struct {
	Handle     string
	GraceTime  time.Duration
	RootFSPath string
	BindMounts []BindMount
	Network    string
}

type BindMount struct {
	SrcPath string
	DstPath string
	Mode    BindMountMode
}

type BindMountMode uint8

const BindMountModeRO BindMountMode = 0
const BindMountModeRW BindMountMode = 1
