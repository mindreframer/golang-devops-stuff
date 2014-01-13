package network_pool

import (
	"fmt"
	"net"
	"sync"

	"github.com/vito/garden/linux_backend/network"
)

type NetworkPool interface {
	Acquire() (*network.Network, error)
	Release(*network.Network)
	Remove(*network.Network) error
	Network() *net.IPNet
}

type RealNetworkPool struct {
	ipNet *net.IPNet

	pool      []*network.Network
	poolMutex *sync.Mutex
}

type PoolExhaustedError struct{}

func (e PoolExhaustedError) Error() string {
	return "network pool is exhausted"
}

type NetworkTakenError struct {
	Network *network.Network
}

func (e NetworkTakenError) Error() string {
	return fmt.Sprintf("network already acquired: %s", e.Network.String())
}

func New(ipNet *net.IPNet) *RealNetworkPool {
	pool := []*network.Network{}

	_, startNet, err := net.ParseCIDR(ipNet.IP.String() + "/30")
	if err != nil {
		panic(err)
	}

	for subnet := startNet; ipNet.Contains(subnet.IP); subnet = nextSubnet(subnet) {
		pool = append(pool, network.New(subnet))
	}

	return &RealNetworkPool{
		ipNet: ipNet,

		pool:      pool,
		poolMutex: new(sync.Mutex),
	}
}

func (p *RealNetworkPool) Acquire() (*network.Network, error) {
	p.poolMutex.Lock()
	defer p.poolMutex.Unlock()

	if len(p.pool) == 0 {
		return nil, PoolExhaustedError{}
	}

	acquired := p.pool[0]
	p.pool = p.pool[1:]

	return acquired, nil
}

func (p *RealNetworkPool) Remove(network *network.Network) error {
	idx := 0
	found := false

	p.poolMutex.Lock()
	defer p.poolMutex.Unlock()

	for i, existingNetwork := range p.pool {
		if existingNetwork.String() == network.String() {
			idx = i
			found = true
			break
		}
	}

	if !found {
		return NetworkTakenError{network}
	}

	p.pool = append(p.pool[:idx], p.pool[idx+1:]...)

	return nil
}

func (p *RealNetworkPool) Release(network *network.Network) {
	if !p.ipNet.Contains(network.IP()) {
		return
	}

	p.poolMutex.Lock()
	defer p.poolMutex.Unlock()

	p.pool = append(p.pool, network)
}

func (p *RealNetworkPool) Network() *net.IPNet {
	return p.ipNet
}

func nextSubnet(ipNet *net.IPNet) *net.IPNet {
	next := net.ParseIP(ipNet.IP.String())

	inc(next)
	inc(next)
	inc(next)
	inc(next)

	_, nextNet, err := net.ParseCIDR(next.String() + "/30")
	if err != nil {
		panic(err)
	}

	return nextNet
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
