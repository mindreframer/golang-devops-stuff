package main

import (
	"encoding/json"
	"github.com/coreos/go-etcd/etcd"
	"github.com/golang/glog"
	"regexp"
	"strings"
)

// A watcher loads and watch the etcd hierarchy for domains and services.
type watcher struct {
	client   *etcd.Client
	config   *Config
	domains  map[string]*Domain
	services map[string]*ServiceCluster
}

// Constructor for a new watcher
func NewEtcdWatcher(config *Config, domains map[string]*Domain, services map[string]*ServiceCluster) (*watcher, error) {
	client, err := config.getEtcdClient()

	if err != nil {
		return nil, err
	}

	return &watcher{client, config, domains, services}, nil
}

//Init domains and services.
func (w *watcher) init() {
	go w.loadAndWatch(w.config.domainPrefix, w.registerDomain)
	go w.loadAndWatch(w.config.servicePrefix, w.registerService)

}

// Loads and watch an etcd directory to register objects like domains, services
// etc... The register function is passed the etcd Node that has been loaded.
func (w *watcher) loadAndWatch(etcdDir string, registerFunc func(*etcd.Node, string)) {
	w.loadPrefix(etcdDir, registerFunc)

	updateChannel := make(chan *etcd.Response, 10)
	go w.watch(updateChannel, registerFunc)
	w.client.Watch(etcdDir, (uint64)(0), true, updateChannel, nil)
}

func (w *watcher) loadPrefix(etcDir string, registerFunc func(*etcd.Node, string)) {
	response, err := w.client.Get(etcDir, true, true)
	if err == nil {
		for _, serviceNode := range response.Node.Nodes {
			registerFunc(serviceNode, response.Action)

		}
	}
}

func (w *watcher) watch(updateChannel chan *etcd.Response, registerFunc func(*etcd.Node, string)) {
	for {
		response := <-updateChannel
		if response != nil {
			registerFunc(response.Node, response.Action)
		}

	}
}

func (w *watcher) registerDomain(node *etcd.Node, action string) {

	domainName := w.getDomainForNode(node)

	domainKey := w.config.domainPrefix + "/" + domainName
	response, err := w.client.Get(domainKey, true, false)

	if action == "delete" || action == "expire" {
		w.RemoveDomain(domainName)
		return
	}

	if err == nil {
		domain := &Domain{}
		for _, node := range response.Node.Nodes {
			switch node.Key {
			case domainKey + "/type":
				domain.typ = node.Value
			case domainKey + "/value":
				domain.value = node.Value
			}
		}

		actualDomain := w.domains[domainName]

		if domain.typ != "" && domain.value != "" && !domain.equals(actualDomain) {
			w.domains[domainName] = domain
			glog.Infof("Registered domain %s with (%s) %s", domainName, domain.typ, domain.value)

		}
	}

}

func (w *watcher) RemoveDomain(key string) {
	delete(w.domains, key)

}

func (w *watcher) getDomainForNode(node *etcd.Node) string {
	r := regexp.MustCompile(w.config.domainPrefix + "/(.*)")
	return strings.Split(r.FindStringSubmatch(node.Key)[1], "/")[0]
}

func (w *watcher) getEnvForNode(node *etcd.Node) string {
	r := regexp.MustCompile(w.config.servicePrefix + "/(.*)(/.*)*")
	return strings.Split(r.FindStringSubmatch(node.Key)[1], "/")[0]
}

func (w *watcher) getEnvIndexForNode(node *etcd.Node) string {
	r := regexp.MustCompile(w.config.servicePrefix + "/(.*)(/.*)*")
	return strings.Split(r.FindStringSubmatch(node.Key)[1], "/")[1]
}

func (w *watcher) RemoveEnv(serviceName string) {
	delete(w.services, serviceName)
}

func (w *watcher) registerService(node *etcd.Node, action string) {
	serviceName := w.getEnvForNode(node)

	// Get service's root node instead of changed node.
	serviceNode, _ := w.client.Get(w.config.servicePrefix+"/"+serviceName, true, true)

	for _, indexNode := range serviceNode.Node.Nodes {

		serviceIndex := w.getEnvIndexForNode(indexNode)
		serviceKey := w.config.servicePrefix + "/" + serviceName + "/" + serviceIndex
		statusKey := serviceKey + "/status"

		response, err := w.client.Get(serviceKey, true, true)

		if err == nil {

			if w.services[serviceName] == nil {
				w.services[serviceName] = &ServiceCluster{}
			}

			service := &Service{}
			service.location = &location{}
			service.index = serviceIndex
			service.nodeKey = serviceKey
			service.name = serviceName

			if action == "delete" {
				glog.Infof("Removing service %s", serviceName)
				w.RemoveEnv(serviceName)
				return
			}

			for _, node := range response.Node.Nodes {
				switch node.Key {
				case serviceKey + "/location":
					location := &location{}
					err := json.Unmarshal([]byte(node.Value), location)
					if err != nil {
						glog.Errorf("Registering service %s has failed - Location is wrong or missing information", serviceName)
						break
					}

					service.location.Host = location.Host
					service.location.Port = location.Port
				case serviceKey + "/domain":
					service.domain = node.Value

				case statusKey:
					service.status = &Status{}
					service.status.service = service
					for _, subNode := range node.Nodes {
						switch subNode.Key {
						case statusKey + "/alive":
							service.status.alive = subNode.Value
						case statusKey + "/current":
							service.status.current = subNode.Value
						case statusKey + "/expected":
							service.status.expected = subNode.Value
						}
					}
				}
			}

			actualEnv := w.services[serviceName].Get(service.index)

			if !actualEnv.equals(service) && service.location != nil {

				if service.location.Host != "" && service.location.Port != 0 {
					w.services[serviceName].Add(service)
					glog.Infof("Registering service %s with location : http://%s:%d/", serviceName, service.location.Host, service.location.Port)

				}

			}
		}
	}
}
