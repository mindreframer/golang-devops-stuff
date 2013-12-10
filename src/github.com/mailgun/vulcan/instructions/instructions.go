package instructions

import (
	"fmt"
	"net/http"
)

// On every request proxy asks control server what to do
// with the request, control server replies with this structure
// or rejects the request.
type ProxyInstructions struct {
	// Allows proxy to fall back to the next upstream
	// if the selected upstream failed
	Failover *Failover
	// Tokens uniquely identify the requester. E.g. token can be account id or
	// combination of ip and account id. Tokens can be throttled as well.
	// The reply can have 0 or several tokens
	Tokens []*Token
	// List of upstreams that can accept this request. Load balancer will
	// choose an upstream based on the algo, e.g. random, round robin,
	// or least connections. At least one upstream is required.
	Upstreams []*Upstream
	// If supplied, headers will be added to the proxied request.
	Headers http.Header
}

func NewProxyInstructions(
	failover *Failover,
	tokens []*Token,
	upstreams []*Upstream,
	headers http.Header) (*ProxyInstructions, error) {

	if len(upstreams) <= 0 {
		return nil, fmt.Errorf("At least one upstream is required")
	}

	return &ProxyInstructions{
		Failover:  failover,
		Tokens:    tokens,
		Upstreams: upstreams,
		Headers:   headers}, nil
}
