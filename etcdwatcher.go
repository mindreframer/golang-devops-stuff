package main

import (
	"github.com/coreos/go-etcd/etcd"
	"log"
	"regexp"
	"strings"
)

type watcher struct {
	client       *etcd.Client
	config       *Config
	domains      map[string]*Domain
	environments map[string]*Environment
}

func NewEtcdWatcher(config *Config, domains map[string]*Domain, envs map[string]*Environment) *watcher {
	client := etcd.NewClient([]string{config.etcdAddress})
	return &watcher{client, config, domains, envs}
}

/**
 * Init domains and environments.
 */
func (w *watcher) init() {
	go w.loadAndWatch(w.config.domainPrefix, w.registerDomain)
	go w.loadAndWatch(w.config.envPrefix, w.registerEnvironment)

}

/**
 * Loads and watch an etcd directory to register objects like domains, environments
 * etc... The register function is passed the etcd Node that has been loaded.
 */
func (w *watcher) loadAndWatch(etcdDir string, registerFunc func(*etcd.Node)) {
	w.loadPrefix(etcdDir, registerFunc)

	updateChannel := make(chan *etcd.Response, 10)
	go w.watch(updateChannel, registerFunc)
	w.client.Watch(etcdDir, (uint64)(0), true, updateChannel, nil)

}

func (w *watcher) loadPrefix(etcDir string, registerFunc func(*etcd.Node)) {
	response, err := w.client.Get(etcDir, true, false)

	if err == nil {
		for _, node := range response.Node.Nodes {
			registerFunc(&node)
		}
	}
}

func (w *watcher) watch(updateChannel chan *etcd.Response, registerFunc func(*etcd.Node)) {
	for {
		response := <-updateChannel
		registerFunc(response.Node)
	}
}

func (w *watcher) registerDomain(node *etcd.Node) {

	domainName := w.getDomainForNode(node)
	domainKey := w.config.domainPrefix + "/" + domainName
	response, err := w.client.Get(domainKey, true, false)

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
		if domain.typ != "" && domain.value != "" {
			w.domains[domainName] = domain
			log.Printf("Registering domain %s with service (%s):%s", domainName, domain.typ, domain.value)
		}
	}

}

func (w *watcher) getDomainForNode(node *etcd.Node) string {
	r := regexp.MustCompile(w.config.domainPrefix + "/(.*)")
	return strings.Split(r.FindStringSubmatch(node.Key)[1], "/")[0]
}

func (w *watcher) getEnvForNode(node *etcd.Node) string {
	r := regexp.MustCompile(w.config.envPrefix + "/(.*)(/.*)*")
	return strings.Split(r.FindStringSubmatch(node.Key)[1], "/")[0]
}

func (w *watcher) registerEnvironment(node *etcd.Node) {
	envName := w.getEnvForNode(node)
	envKey := w.config.envPrefix + "/" + envName
	statusKey := w.config.envPrefix + "/" + envName + "/status"

	response, err := w.client.Get(envKey, true, true)

	if err == nil {
		env := &Environment{}
		for _, node := range response.Node.Nodes {
			switch node.Key {
			case envKey + "/ip":
				env.ip = node.Value
			case envKey + "/port":
				env.port = node.Value
			case envKey + "/domain":
				env.domain = node.Value
			case statusKey:
			  env.status = &Status{}
			  for _, subNode := range node.Nodes {
				  switch subNode.Key {
					case statusKey + "/alive":
						env.status.alive = subNode.Value
					case statusKey + "/current":
						env.status.current = subNode.Value
					case statusKey + "/expected":
						env.status.expected = subNode.Value
					}
				}
			}
		}
		if env.ip != "" && env.port != "" {
			w.environments[envName] = env
			log.Printf("Registering environment %s with address : http://%s:%s/", envName, env.ip, env.port)
			if env.domain != "" && w.domains[env.domain] != nil {
				w.domains[env.domain].server = nil
				log.Printf("Reset domain %s", env.domain)
			}
		}
		if env.status != nil && env.status.current != "" {
			w.environments[envName] = env
			log.Printf("Watching environment %s status : Alive: %s - Current: %s - Expected: %s", envName, env.status.alive, env.status.current, env.status.expected)
		}
	}
}
