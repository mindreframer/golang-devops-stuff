package network

import (
	"github.com/stripe-ctf/octopus/state"
)

type Network struct {
	links []*Link
	// This just slices into links, the canonical data source. It exists to
	// make finding specific links fast
	cache [][]*Link
}

func New() *Network {
	count := uint(state.NodeCount())
	net := &Network{
		links: make([]*Link, 0),
		cache: make([][]*Link, count),
	}

	var i, j, offset uint
	for i = 0; i < count; i++ {
		for j = 0; j < i; j++ {
			link := &Link{
				network: net,
				kill:    make(chan bool),
				// Do smaller number as agent1
				agent1: j,
				agent2: i,
			}
			net.links = append(net.links, link)
		}
	}

	for offset, i = 0, 0; i < count; offset, i = offset+i, i+1 {
		net.cache[i] = net.links[offset : offset+i]
	}

	return net
}

func (n *Network) Start() {
	for _, link := range n.links {
		link.Listen()
	}
}

func (n *Network) Link(agent1, agent2 uint) *Link {
	nservers := uint(len(n.cache))
	if agent1 == agent2 || agent1 >= nservers || agent2 >= nservers {
		return nil
	} else if agent1 > agent2 {
		return n.Link(agent2, agent1)
	} else {
		return n.cache[agent2][agent1]
	}
}

func (n *Network) Links() []*Link {
	return n.links
}

// Find the perimeter links of a given set of nodes. Most useful for determining
// which nodes lie along a netsplit
func (n *Network) FindPerimeter(servers []uint) []*Link {
	set := make([]bool, len(n.cache))
	for _, id := range servers {
		set[id] = true
	}

	perimeter := make([]*Link, 0, len(servers)*(len(n.cache)-len(servers)))
	for i := 0; i < len(n.links); i++ {
		if set[n.links[i].agent1] != set[n.links[i].agent2] {
			perimeter = append(perimeter, n.links[i])
		}
	}

	return perimeter
}
