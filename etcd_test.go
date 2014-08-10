package main

import (
  "github.com/coreos/go-etcd/etcd"
  "testing"
)

func TestEtcd_shouldValidateARecords(t *testing.T) {

  aRecordNode := &etcd.Node{ Key: "/example/A", Value: "notanip" }

  valid, _ := validate(aRecordNode)

  if valid {
    t.Errorf("Should not accept an invalid ip for A record")
  }

}

func TestEtcd_shouldValidateCNAMERecords(t *testing.T) {

  cnameRecordNode := &etcd.Node{ Key: "/example/CNAME", Value: "notfqdn" }

  valid, _ := validate(cnameRecordNode)

  if valid {
    t.Errorf("Should not accept a non-fqdn value for a CNAME record")
  }

}

func TestEtcd_shouldValidatePTRRecords(t *testing.T) {

  ptrRecordNode := &etcd.Node{ Key: "/example/PTR", Value: "notfqdn" }

  valid, _ := validate(ptrRecordNode)

  if valid {
    t.Errorf("Should not accept a non-fqdn value for a PTR record")
  }

}
