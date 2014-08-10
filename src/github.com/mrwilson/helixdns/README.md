## HelixDNS

 [![Build Status](https://travis-ci.org/mrwilson/helixdns.png?branch=master)](https://travis-ci.org/mrwilson/helixdns)

  A simple dns server to read records from etcd. See this [blog post](http://probablyfine.co.uk/2014/03/02/serving-dns-records-from-etcd/) for more information.

## Build Instructions

    go get github.com/mrwilson/helixdns

    go build -o hdns github.com/mrwilson/helixdns

## Uses

    hdns
      -port=<port-to-run-on>
      -etcd-address=<address-of-etcd-instance>
      -forward=<address-of-forwarding-nameserver>

## SRV Records

  SRV records have multiple pieces of information to serve, so the values stored in etcd under `/helix/com/example/_<protocol>/_<service>/SRV` should be in the form of a list of JSON objects, as below.

    [
      {"Priority":10,"Weight":60,"Port":5060,"Target":"bigbox.example.com."},
      {"Priority":10,"Weight":20,"Port":5060,"Target":"smallbox1.example.com."},
      {"Priority":10,"Weight":10,"Port":5060,"Target":"smallbox2.example.com."},
      {"Priority":10,"Weight":10,"Port":5066,"Target":"smallbox2.example.com."},
      {"Priority":20,"Weight":0, "Port":5060,"Target":"backupbox.example.com."}
    ]

## TODO

 * Other types of record that aren't A, AAAA, CNAME, PTR, or SRV.
