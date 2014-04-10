package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"html/template"
)

var page = `<html>
  <body>
    {{template "content" .Content}}
  </body>
</html>`

var content = `{{define "content"}}
<div>
   <p>{{.Title}}</p>
   <p>{{.Content}}</p>
</div>
{{end}}`

type Content struct {
   Title string
   Content string
}

type Page struct {
    Content *Content
}

type domainResolver interface {
	resolve(domain string) (http.Handler, bool)
	redirectToStatusPage(domainName string) (string)
	init()
}

type proxy struct {
	config         *Config
	domainResolver domainResolver
}

func NewProxy(c *Config, resolver domainResolver) *proxy {
	return &proxy{c, resolver}
}

func (p *proxy) start() {
	log.Printf("Listening on port %d", p.config.port)
	http.HandleFunc("/", p.OnRequest)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", p.config.port), nil))
}

func (p *proxy) OnRequest(w http.ResponseWriter, r *http.Request) {
	host := hostnameOf(r.Host)
	// Check if host is in pending, stopping or error state
	redirect := p.domainResolver.redirectToStatusPage(host)
	if redirect != "" {
		pagedata := &Page{Content: &Content{Title:"Status", Content:redirect}}
		tmpl, err := template.New("page").Parse(page)
    tmpl, err = tmpl.Parse(content)
		if(err==nil){
			tmpl.Execute(w, pagedata)
		}
		return
	}

	server, found := p.domainResolver.resolve(host)

	if found {
		server.ServeHTTP(w, r)
		return
	}

	http.NotFound(w, r)
}

func hostnameOf(host string) string {
	hostname := strings.Split(host, ":")[0]

	if len(hostname) > 4 && hostname[0:4] == "www." {
		hostname = hostname[4:]
	}

	return hostname
}
