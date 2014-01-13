package command

import (
	"github.com/mailgun/vulcan/loadbalance"
)

type Endpoint struct {
	Upstream *Upstream
	Eid      string
	Active   bool
}

func NewEndpoint(upstream *Upstream, active bool) *Endpoint {
	return &Endpoint{
		Eid:      upstream.Id,
		Upstream: upstream,
		Active:   active,
	}
}

func (e *Endpoint) Id() string {
	return e.Eid
}

func (e *Endpoint) IsActive() bool {
	return e.Active
}

func EndpointsFromUpstreams(upstreams []*Upstream) []loadbalance.Endpoint {
	endpoints := make([]loadbalance.Endpoint, len(upstreams))
	for i, upstream := range upstreams {
		endpoints[i] = NewEndpoint(upstream, true)
	}
	return endpoints
}
