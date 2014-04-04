package helixdns

import (
  "github.com/miekg/dns"
  "testing"
)

var etcdAddress = ""

type FakeClient struct {}

func (f FakeClient) Get(address string) (Response, error) {
  etcdAddress = address
  return nil, nil
}

func TestServer_looksUpCorrectEntry(t *testing.T) {

  client   := &FakeClient{}
  server   := &HelixServer{ Port: 1000, Client: client }
  question := &dns.Question{ Name: "foo.example.com.", Qtype: 1}

  server.getResponse(*question)

  if etcdAddress != "helix/com/example/foo/A" {
    t.Errorf("Incorrect address: %s", etcdAddress)
  }

}
