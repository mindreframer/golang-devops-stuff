package fake_network_pool

import (
	"net"

	"github.com/vito/garden/linux_backend/network"
)

type FakeNetworkPool struct {
	ipNet       *net.IPNet
	nextNetwork net.IP

	AcquireError error
	RemoveError  error

	Released []string
	Removed  []string
}

func New(ipNet *net.IPNet) *FakeNetworkPool {
	return &FakeNetworkPool{
		ipNet: ipNet,

		nextNetwork: ipNet.IP,
	}
}

func (p *FakeNetworkPool) Acquire() (*network.Network, error) {
	if p.AcquireError != nil {
		return nil, p.AcquireError
	}

	_, ipNet, err := net.ParseCIDR(p.nextNetwork.String() + "/30")
	if err != nil {
		return nil, err
	}

	inc(p.nextNetwork)
	inc(p.nextNetwork)
	inc(p.nextNetwork)
	inc(p.nextNetwork)

	return network.New(ipNet), nil
}

func (p *FakeNetworkPool) Remove(network *network.Network) error {
	if p.RemoveError != nil {
		return p.RemoveError
	}

	p.Removed = append(p.Removed, network.String())

	return nil
}

func (p *FakeNetworkPool) Release(network *network.Network) {
	p.Released = append(p.Released, network.String())
}

func (p *FakeNetworkPool) Network() *net.IPNet {
	return p.ipNet
}

func inc(ip net.IP) {
	for j := len(ip) - 1; j >= 0; j-- {
		ip[j]++
		if ip[j] > 0 {
			break
		}
	}
}
