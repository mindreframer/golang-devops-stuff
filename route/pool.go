package route

import (
	"encoding/json"
	"math/rand"
)

type Pool struct {
	endpoints map[string]*Endpoint
}

func NewPool() *Pool {
	return &Pool{
		endpoints: make(map[string]*Endpoint),
	}
}

func (p *Pool) Add(endpoint *Endpoint) {
	p.endpoints[endpoint.CanonicalAddr()] = endpoint
}

func (p *Pool) Remove(endpoint *Endpoint) {
	delete(p.endpoints, endpoint.CanonicalAddr())
}

func (p *Pool) Sample() (*Endpoint, bool) {
	if len(p.endpoints) == 0 {
		return nil, false
	}

	index := rand.Intn(len(p.endpoints))

	ticker := 0
	for _, endpoint := range p.endpoints {
		if ticker == index {
			return endpoint, true
		}

		ticker += 1
	}

	panic("unreachable")
}

func (p *Pool) FindByPrivateInstanceId(id string) (*Endpoint, bool) {
	for _, endpoint := range p.endpoints {
		if endpoint.PrivateInstanceId == id {
			return endpoint, true
		}
	}

	return nil, false
}

func (p *Pool) IsEmpty() bool {
	return len(p.endpoints) == 0
}

func (p *Pool) MarshalJSON() ([]byte, error) {
	addresses := []string{}

	for addr, _ := range p.endpoints {
		addresses = append(addresses, addr)
	}

	return json.Marshal(addresses)
}
