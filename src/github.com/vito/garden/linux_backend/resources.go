package linux_backend

import (
	"sync"

	"github.com/vito/garden/linux_backend/network"
)

type Resources struct {
	UID     uint32
	Network *network.Network
	Ports   []uint32

	portsLock *sync.Mutex
}

func NewResources(
	uid uint32,
	network *network.Network,
	ports []uint32,
) *Resources {
	return &Resources{
		UID:     uid,
		Network: network,
		Ports:   ports,

		portsLock: new(sync.Mutex),
	}
}

func (r *Resources) AddPort(port uint32) {
	r.portsLock.Lock()
	defer r.portsLock.Unlock()

	r.Ports = append(r.Ports, port)
}
