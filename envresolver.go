package main

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type EnvResolver struct {
	config          *Config
	watcher         *watcher
	services            map[string]*ServiceCluster
	dest2ProxyCache map[string]http.Handler
}

func NewEnvResolver(c *Config) *EnvResolver {
	services := make(map[string]*ServiceCluster)
	w,_ := NewEtcdWatcher(c, nil, services)
	return &EnvResolver{c, w, services, make(map[string]http.Handler)}
}

func (r *EnvResolver) resolve(domain string) (http.Handler, error) {
	serviceName := strings.Split(domain, ".")[0]

	serviceTree := r.services[serviceName]
	if serviceTree != nil {

		if service, err := serviceTree.Next(); err != nil {
			uri := fmt.Sprintf("http://%s:%d/", service.location.Host, service.location.Port)
			return r.getOrCreateProxyFor(uri), nil
		}
	}

	return nil, errors.New("Unable to resolve")

}

func (r *EnvResolver) init() {
	r.watcher.loadAndWatch(r.config.servicePrefix, r.watcher.registerService)
}

func (r *EnvResolver) redirectToStatusPage(domainName string) string {
	return ""
}

func (r *EnvResolver) getOrCreateProxyFor(uri string) http.Handler {
	if _, ok := r.dest2ProxyCache[uri]; !ok {
		dest, _ := url.Parse(uri)
		r.dest2ProxyCache[uri] = httputil.NewSingleHostReverseProxy(dest)
	}
	return r.dest2ProxyCache[uri]
}
