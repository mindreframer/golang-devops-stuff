package instructions

import (
	"github.com/mailgun/vulcan/netutils"
	"net/url"
)

// Upstream is HTTP server that will actually serve
// the request that would be proxied
type Upstream struct {
	// URL of the upstream, would be used for throttling
	Url *url.URL
	// Upstreams can be rate controlled, if at least one rate
	// does not allow upstream it won't be chosen by the load balancer
	Rates []*Rate
	// Every upstream can supply the headers to add to the request
	// in case if the upsteam has been selected by the load balancer
	Headers map[string][]string
}

func NewUpstream(inUrl string, rates []*Rate, headers map[string][]string) (*Upstream, error) {
	//To ensure that upstream is correct url
	parsedUrl, err := netutils.ParseUrl(inUrl)
	if err != nil {
		return nil, err
	}

	return &Upstream{
		Url:     parsedUrl,
		Rates:   rates,
		Headers: headers,
	}, nil
}

func (upstream *Upstream) Id() string {
	url := &url.URL{
		Scheme: upstream.Url.Scheme,
		Host:   upstream.Url.Host,
	}
	return url.String()
}

func (upstream *Upstream) String() string {
	return upstream.Url.String()
}
