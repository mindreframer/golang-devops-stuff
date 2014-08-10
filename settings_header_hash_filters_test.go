package main

import (
	"testing"
	"net/http"
)

func TestHTTPHeaderHashFilters(t *testing.T) {
	filters := HTTPHeaderHashFilters{}

	err := filters.Set("Header1:1/2")
	if err != nil {
		t.Error("Should not error on Header1:^$")
	}

	err = filters.Set("Header2:1/2")
	if err != nil {
		t.Error("Should not error on Header2:^:$")
	}

	err = filters.Set("HeaderIrrelevant:1/3")
	if err == nil {
		t.Error("Should error on HeaderIrrelevant:1/3")
	}

	req := http.Request{}
	req.Header = make(map[string][]string)
	req.Header.Add("Header1", "test3414")

	if(filters.Good(&req)) {
		t.Error("Request should not pass filters, Header2 does not exist")
	}

	req.Header.Add("Header2", "test2")
	if(filters.Good(&req)) {
		t.Error("Request should not pass filters, Header2 hash too high")
	}

	req.Header.Set("Header2", "test3414")
	if(!filters.Good(&req)) {
		t.Error("Request should pass filters")
	}
}
