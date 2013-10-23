package replay

import (
	"flag"
	"reflect"
	"testing"
)

func TestAddress(t *testing.T) {
	settings := &ReplaySettings{
		Host: "local",
		Port: 2,
	}

	settings.Parse()

	if settings.Address != "local:2" {
		t.Error("Address not match")
	}
}

func TestForwardAddress(t *testing.T) {
	settings := &ReplaySettings{
		Host:           "local",
		Port:           2,
		ForwardAddress: "host1:1,host2:2|10",
	}

	settings.Parse()

	forward_hosts := settings.ForwardedHosts()

	if len(forward_hosts) != 2 {
		t.Error("Should have 2 forward hosts")
	}

	if forward_hosts[0].Limit != 0 && forward_hosts[0].Url != "host1:1" {
		t.Error("Host should be host1:1 with no limit")
	}

	if forward_hosts[1].Limit != 10 && forward_hosts[0].Url != "host2:2" {
		t.Error("Host should be host2:2 with 10 limit")
	}
}

func TestElasticSearchSettings(t *testing.T) {
	settings := &ReplaySettings{
		Host:            "local",
		Port:            2,
		ForwardAddress:  "host1:1,host2:2|10",
		ElastiSearchURI: "host:10/index_name",
	}

	// FIXME: This is redundant. We could assign `Settings = *settings` to
	// check the code path in Init(), but it would result in the ES plugin
	// being registered twice.
	settings.Parse()

	esp := &ESPlugin{}
	esp.Init(settings.ElastiSearchURI)

	if esp.ApiPort != "10" {
		t.Error("Port not match")
	}

	if esp.Host != "host" {
		t.Error("Host not match")
	}

	if esp.Index != "index_name" {
		t.Error("Index not match")
	}
}

func TestAdditionalHeaders(t *testing.T) {
	args := []string{
		"-header", "Empty:",
		"-header", "Foo: contains:multiple:colons",
		"-header", "Host:nospaces.example.com",
		"-header", "Authorization: Basic Zm9vOmJhcg==",
		"-header", " User-Agent :  Contains leading and trailing space ",
	}
	out := Headers{
		{"Empty", ""},
		{"Foo", "contains:multiple:colons"},
		{"Host", "nospaces.example.com"},
		{"Authorization", "Basic Zm9vOmJhcg=="},
		{"User-Agent", "Contains leading and trailing space"},
	}

	headers := Headers{}
	fs := flag.NewFlagSet("TestHeaders", flag.ExitOnError)
	fs.Var(&headers, "header", "blah")
	fs.Parse(args)

	if !reflect.DeepEqual(headers, out) {
		t.Error("Headers not parsed as expected")
	}

	args = []string{
		"-header", "contains no colons",
	}
	headers = Headers{}
	fs = flag.NewFlagSet("TestHeaders", flag.ContinueOnError)
	fs.Var(&headers, "header", "blah")
	err := fs.Parse(args)

	if err == nil {
		t.Error("Invalid header should be rejected")
	}
}
