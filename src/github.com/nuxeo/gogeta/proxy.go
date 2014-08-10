package main

import (
	"fmt"
	"net/http"
	"strings"
	"github.com/golang/glog"
)

type domainResolver interface {
	resolve(domain string) (http.Handler, error)
	init()
}

type proxy struct {
	config         *Config
	domainResolver domainResolver
}

func NewProxy(c *Config, resolver domainResolver) *proxy {
	return &proxy{c, resolver}
}

type proxyHandler func(http.ResponseWriter, *http.Request) (*Config, error)

func (ph proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	defer func() {
		if r := recover(); r != nil {
			http.Error(w, "An error occured serving request", 500)
			glog.Errorf("Recovered from error : %s", r)
		}
	}()

	if c, err := ph(w, r); err != nil {
		ph.OnError(w, r, err, c)
	}
}

func (ph proxyHandler) OnError(w http.ResponseWriter, r *http.Request, error error, c *Config) {
	if stError, ok := error.(StatusError); ok {
		sp := &StatusPage{c, stError}
		// Check if status is passivated -> setting expected state = started
		if sp.error.computedStatus == PASSIVATED_STATUS {
			reactivate(sp, c)
		}
		sp.serve(w, r)
	} else {
		sp := &StatusPage{c, StatusError{"notfound", nil}}
		sp.serve(w, r)
	}
}



func (p *proxy) start() {
	glog.Infof("Listening on port %d", p.config.port)
	http.Handle("/__static__/", http.FileServer(http.Dir(p.config.templateDir)))
	http.Handle("/", proxyHandler(p.proxy))
	glog.Fatalf("%s", http.ListenAndServe(fmt.Sprintf(":%d", p.config.port), nil))

}


func (p *proxy) proxy(w http.ResponseWriter, r *http.Request) (*Config, error) {

	if p.config.forceFwSsl &&  "https" != r.Header.Get("x-forwarded-proto") {

		http.Redirect(w, r, fmt.Sprintf("https://%s%s", hostnameOf(r.Host), r.URL.String() ), http.StatusMovedPermanently)
		return p.config, nil
	}


	host := hostnameOf(r.Host)
	if server, err := p.domainResolver.resolve(host); err != nil {
		return p.config, err
	} else {
		server.ServeHTTP(w, r)
		return p.config, nil
	}
}

func hostnameOf(host string) string {
	hostname := strings.Split(host, ":")[0]

	if len(hostname) > 4 && hostname[0:4] == "www." {
		hostname = hostname[4:]
	}

	return hostname
}

func reactivate(sp *StatusPage, c *Config) {
	client, _ := c.getEtcdClient()
	_, error := client.Set(c.servicePrefix+"/"+sp.error.status.service.name+"/"+sp.error.status.service.index+"/status/expected", STARTED_STATUS, 0)
	if (error != nil) {
		glog.Errorf("Fail: setting expected state = 'started' for instance %s. Error:%s", sp.error.status.service.name, error)
	}
	glog.Infof("Instance %s is ready for re-activation", sp.error.status.service.name)
}
