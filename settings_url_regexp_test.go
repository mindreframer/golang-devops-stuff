package main

import (
	"testing"
	"net/http"
	"net/url"
)

func TestHTTPUrlRegexp(t *testing.T) {
	filter := HTTPUrlRegexp{}

	filter.Set("^www.google.com/admin/")

	req := http.Request{}
	req.Host = "www.google.com"
	var err error
	req.URL, err = url.Parse("/admin/testpage1")
	if(!filter.Good(&req) || err != nil) {
		t.Error("Request should pass filters")
	}

	req.URL, err = url.Parse("/user/testpage2")
	if(filter.Good(&req) || err != nil) {
		t.Error("Request should not pass filters")
	}
}
