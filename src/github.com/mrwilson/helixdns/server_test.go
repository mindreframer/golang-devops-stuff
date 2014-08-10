package main

import (
  "github.com/miekg/dns"
  "testing"
)

type FakeClient struct {
  Result string
}

func (f *FakeClient) Get(address string) (Response, error) {
  f.Result = address
  return EtcdResponse{}, nil
}

func (f FakeClient) WatchForChanges() {}

func TestServer_looksUpCorrectEntry(t *testing.T) {

  client   := &FakeClient{ Result: "" }
  server   := &HelixServer{ Port: 1000, Client: client }
  question := &dns.Question{ Name: "foo.example.com.", Qtype: 1}

  response, _ := server.getResponse(*question)

  if response == nil {
    t.Errorf("Returned nil for etcd lookup")
  }

  if client.Result != "helix/com/example/foo/A" {
    t.Errorf("Incorrect address: %s", client.Result)
  }

}
