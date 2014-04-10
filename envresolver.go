package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
)

type EnvResolver struct {
	config  *Config
	watcher *watcher
	envs    map[string]*Environment
}

func NewEnvResolver(c *Config) *EnvResolver {
	envs := make(map[string]*Environment)
	w := NewEtcdWatcher(c, nil, envs)
	return &EnvResolver{c, w, envs}
}

func (r *EnvResolver) resolve(domain string) (http.Handler, bool) {
	envName := strings.Split(domain, ".")[0]

	env := r.envs[envName]
	if env != nil {
		if env.server == nil {
			uri := ""
			if env.port != "80" {
				uri = fmt.Sprintf("http://%s:%s/", env.ip, env.port)

			} else {
				uri = fmt.Sprintf("http://%s/", env.ip)
			}
			dest, _ := url.Parse(uri)
			env.server = httputil.NewSingleHostReverseProxy(dest)
		}
		return env.server, true
	}

	return nil, false

}

func (r *EnvResolver) init() {
	r.watcher.loadAndWatch(r.config.envPrefix, r.watcher.registerEnvironment)
}

func (r *EnvResolver) redirectToStatusPage(domainName string) (string){
	return ""
}
