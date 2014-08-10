package main

import (
	"net/http"
	"net/http/httputil"
	"net/url"
)

type DummyResolver struct {
}

func (*DummyResolver) resolve(domain string) (http.Handler, error) {
	dest, _ := url.Parse("http://localhost:8080/")
	return httputil.NewSingleHostReverseProxy(dest), nil
}

func (*DummyResolver) init() {

}

func (*DummyResolver) redirectToStatusPage(domainName string) string {
	return ""
}
