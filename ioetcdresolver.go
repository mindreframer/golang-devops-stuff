package main

import (
	"fmt"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
)

const (
		STARTING_STATUS = "starting"
		STARTED_STATUS = "started"
		STOPPING_STATUS = "stopping"
		STOPPED_STATUS = "stopped"
		ERROR_STATUS = "error"
		NA_STATUS = "n/a"
		STOPPED_STATUS_PAGE = "Stopped!"
		STARTING_STATUS_PAGE = "Starting..."
		ERROR_STATUS_PAGE = "Error!"
)

type Domain struct {
	typ    string
	value  string
	server http.Handler
}

type Environment struct {
	ip     string
	port   string
	domain string
	server http.Handler
	status *Status
}

type Status struct {
	alive string
	current string
	expected string
}

type IoEtcdResolver struct {
	config       *Config
	watcher      *watcher
	domains      map[string]*Domain
	environments map[string]*Environment
}

func NewEtcdResolver(c *Config) *IoEtcdResolver {
	domains := make(map[string]*Domain)
	envs := make(map[string]*Environment)
	w := NewEtcdWatcher(c, domains, envs)
	return &IoEtcdResolver{c, w, domains, envs}
}

func (r *IoEtcdResolver) init() {
	r.watcher.init()
}

func (r *IoEtcdResolver) resolve(domainName string) (http.Handler, bool) {
	domain := r.domains[domainName]
	if domain != nil {
		if domain.server == nil {
			log.Printf("Building new HostReverseProxy for %s", domainName)
			switch domain.typ {
			case "iocontainer":
				env := r.environments[domain.value]
				uri := ""
				if env.port != "80" {
					uri = fmt.Sprintf("http://%s:%s/", env.ip, env.port)

				} else {
					uri = fmt.Sprintf("http://%s/", env.ip)
				}
				dest, _ := url.Parse(uri)
				domain.server = httputil.NewSingleHostReverseProxy(dest)

			case "uri":
				dest, _ := url.Parse(domain.value)
				domain.server = httputil.NewSingleHostReverseProxy(dest)
			}
		}

		return domain.server, true
	}
	return nil, false
}

func (r *IoEtcdResolver) redirectToStatusPage(domainName string) (string){
	domain := r.domains[domainName]
	if domain != nil && domain.typ == "iocontainer" {
		env := r.environments[domain.value]
		if env.status != nil {
			alive := env.status.alive
			expected := env.status.expected
			current := env.status.current
			switch current {
				case STOPPED_STATUS:
					if expected==STOPPED_STATUS {
						return STOPPED_STATUS_PAGE
					} else {
						return ERROR_STATUS_PAGE
					}
				case STARTING_STATUS:
					if expected==STARTED_STATUS {
						return STARTING_STATUS_PAGE
					} else {
						return ERROR_STATUS_PAGE
					}
				case STARTED_STATUS:
					if alive!="" {
							if expected!=STARTED_STATUS {
								return ERROR_STATUS_PAGE
							}
					} else {
						return ERROR_STATUS_PAGE
					}
				case STOPPING_STATUS:
					if expected==STOPPED_STATUS {
						return STOPPED_STATUS_PAGE
					} else {
						return ERROR_STATUS_PAGE
					}
				// N/A
				default:
				  return ""
			}
		}
	}
	return ""
}
