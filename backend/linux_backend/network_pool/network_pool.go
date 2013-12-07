package network_pool

import (
	"net"
	"sync"

	"github.com/vito/garden/backend/linux_backend/network"
)

type NetworkPool interface {
	Acquire() (network.Network, error)
	Release(network.Network)
	Network() *net.IPNet
}

type RealNetworkPool struct {
	ipNet *net.IPNet

	pool []*Network

	sync.Mutex
}

type Network struct {
	ipNet *net.IPNet

	hostIP      net.IP
	containerIP net.IP
}

func (n Network) String() string {
	return n.ipNet.String()
}

func (n Network) HostIP() net.IP {
	return n.hostIP
}

func (n Network) ContainerIP() net.IP {
	return n.containerIP
}

type PoolExhaustedError struct{}

func (e PoolExhaustedError) Error() string {
	return "Network pool is exhausted"
}

func New(ipNet *net.IPNet) *RealNetworkPool {
	pool := []*Network{}

	_, startNet, err := net.ParseCIDR(ipNet.IP.String() + "/30")
	if err != nil {
		panic(err)
	}

	for subnet := startNet; ipNet.Contains(subnet.IP); subnet = nextSubnet(subnet) {
		pool = append(pool, networkFor(subnet))
	}

	return &RealNetworkPool{
		ipNet: ipNet,

		pool: pool,
	}
}

func (p *RealNetworkPool) Acquire() (network.Network, error) {
	p.Lock()
	defer p.Unlock()

	if len(p.pool) == 0 {
		return nil, PoolExhaustedError{}
	}

	acquired := p.pool[0]
	p.pool = p.pool[1:]

	return acquired, nil
}

func (p *RealNetworkPool) Release(n network.Network) {
	network := n.(*Network)

	if !p.ipNet.Contains(network.ipNet.IP) {
		return
	}

	p.Lock()
	defer p.Unlock()

	p.pool = append(p.pool, network)
}

func (p *RealNetworkPool) Network() *net.IPNet {
	return p.ipNet
}

func networkFor(ipNet *net.IPNet) *Network {
	return &Network{
		ipNet:       ipNet,
		hostIP:      nextIP(ipNet.IP),
		containerIP: nextIP(nextIP(ipNet.IP)),
	}
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

func nextIP(ip net.IP) net.IP {
	next := net.ParseIP(ip.String())
	inc(next)
	return next
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
