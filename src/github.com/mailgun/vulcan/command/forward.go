package command

import (
	"fmt"
	"net/http"
)

// Forward reply tells proxy to forward the request to the given upstreams
type Forward struct {
	// Allows proxy to fall back to the next upstream
	// if the selected upstream failed
	Failover *Failover
	// Each rate key represents a resource to limit, e.g. ip address or account id
	// or a combination of both. Key is a string with a list of rates attached to it.
	// If any of the given rates of the reuqest is exceeded, the request will be not allowed.
	Rates map[string][]*Rate
	// List of upstreams that can accept this request. Load balancer will
	// choose an upstream based on the algo, e.g. random, round robin,
	// or least connections. At least one upstream is required.
	Upstreams []*Upstream
	// If supplied, headers will be added to the proxied request.
	AddHeaders http.Header
	// These headers will be removed from the request header if met, note that
	// hop headers will be removed no matter what.
	RemoveHeaders []string
	// If not emtpy, the original request path will be replaced with the supplied path
	RewritePath string
}

func NewForward(
	failover *Failover,
	rates map[string][]*Rate,
	upstreams []*Upstream,
	addHeaders http.Header,
	removeHeaders []string) (*Forward, error) {

	if len(upstreams) <= 0 {
		return nil, fmt.Errorf("At least one upstream is required")
	}

	return &Forward{
		Failover:      failover,
		Rates:         rates,
		Upstreams:     upstreams,
		AddHeaders:    addHeaders,
		RemoveHeaders: removeHeaders,
	}, nil
}

func NewForwardFromDict(in map[string]interface{}) (interface{}, error) {
	upstreamsI, exists := in["upstreams"]
	if !exists {
		return nil, fmt.Errorf("Upstreams are required")
	}

	var upstreams []*Upstream
	var err error

	switch upstreamsI.(type) {
	case []string:
		upstreams, err = NewUpstreamsFromUrls(upstreamsI.([]string))
	default:
		upstreams, err = NewUpstreamsFromObj(upstreamsI)
	}

	if err != nil {
		return nil, err
	}

	ratesI, exists := in["rates"]
	var rates map[string][]*Rate
	if exists {
		rates, err = NewRatesFromObj(ratesI)
		if err != nil {
			return nil, err
		}
	}

	failoverI, exists := in["failover"]
	var failover *Failover
	if exists {
		failover, err = NewFailoverFromObj(failoverI)
		if err != nil {
			return nil, err
		}
	}

	pathI, exists := in["rewrite_path"]
	ok := false
	rewritePath := ""
	if exists {
		rewritePath, ok = pathI.(string)
		if !ok {
			return nil, fmt.Errorf("Rewrite-path should be a string")
		}
	}

	addHeaders, removeHeaders, err := AddRemoveHeadersFromDict(in)
	if err != nil {
		return nil, err
	}

	return &Forward{
		Rates:         rates,
		Failover:      failover,
		Upstreams:     upstreams,
		AddHeaders:    addHeaders,
		RemoveHeaders: removeHeaders,
		RewritePath:   rewritePath,
	}, nil
}
