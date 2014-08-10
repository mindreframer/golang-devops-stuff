package main

import (
  "github.com/miekg/dns"
)

type DNSClient interface {
  Lookup(*dns.Msg) (*dns.Msg, error)
  GetAddress() string
}

type ForwardingDNSClient struct {
  Address string
}

func (c ForwardingDNSClient) GetAddress() string {
  return c.Address
}

func (c ForwardingDNSClient) Lookup(req *dns.Msg) (*dns.Msg, error)  {
  return dns.Exchange(req, c.Address)
}
