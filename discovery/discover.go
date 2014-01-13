package discovery

import (
	"fmt"
	"github.com/golang/glog"
	"net/url"
	"strings"
)

type Service interface {
	Get(key string) ([]string, error)
}

type NoopDiscovery struct{}

func NewNoopDiscovery() *NoopDiscovery {
	return &NoopDiscovery{}
}

func (d *NoopDiscovery) Get(serviceName string) ([]string, error) {
	return []string{}, nil
}

func New(discoveryUrl string) (Service, error) {

	if !strings.Contains(discoveryUrl, ":") {
		discoveryUrl = discoveryUrl + "://"
	}

	u, err := url.Parse(discoveryUrl)

	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "noop":
		return NewNoopDiscovery(), nil
	case "disabled":
		return NewNoopDiscovery(), nil
	case "rackspace":
		return NewRackspaceFromUrl(u)
	case "etcd":
		hosts := strings.Split(u.Host, ",")
		return NewEtcd(hosts), nil
	default:
		glog.Errorf("Bad URL for discovery: %s", discoveryUrl)
		return nil, fmt.Errorf("invalid configuration: Unknown discovery scheme: %s", u.Scheme)
	}
}
