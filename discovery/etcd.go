package discovery

import (
	"github.com/coreos/go-etcd/etcd"
	"github.com/golang/glog"
)

type Etcd struct {
	client *etcd.Client
	cache  map[string][]string
}

func NewEtcd(machines []string) *Etcd {
	glog.Infof("Initialized etcd discovery service: %v", machines)

	service := &Etcd{
		client: etcd.NewClient(machines),
		cache:  make(map[string][]string),
	}

	service.UpdateCache()

	go service.Watch()

	return service
}

func (e *Etcd) Watch() {
	for {
		_, err := e.client.Watch("/", 0, true, nil, nil)
		if err != nil {
			glog.Errorf("Error watching for a change: %v", err)
			continue
		}
		glog.Infof("Etcd update detected")
		e.UpdateCache()
	}
}

func (e *Etcd) UpdateCache() {
	glog.Infof("Updating the cache")

	response, err := e.client.Get("/", false, true)

	if err != nil {
		glog.Errorf("Error updating cache: %v", err)
	}

	for _, node := range response.Node.Nodes {
		serviceName := node.Key

		upstreams := make([]string, len(node.Nodes))
		for i, node := range node.Nodes {
			upstreams[i] = node.Value
		}

		e.cache[serviceName] = upstreams
	}

	glog.Infof("Updated cache: %v", e.cache)
}

func (e *Etcd) Get(serviceName string) ([]string, error) {
	if upstreams, ok := e.cache[serviceName]; ok {
		glog.Infof("Found upstreams: %v %v", serviceName, upstreams)
		return upstreams, nil
	}

	glog.Infof("Not found upstreams: %v", serviceName)
	return []string{}, nil
}
