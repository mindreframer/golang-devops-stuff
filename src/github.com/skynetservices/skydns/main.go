// Copyright (c) 2013 The SkyDNS Authors. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"github.com/goraft/raft"
	"github.com/miekg/dns"
	"github.com/skynetservices/skydns/server"
	"github.com/skynetservices/skydns/stats"
	"log"
	"net"
	"os"
	"strings"
	"time"
)

var (
	join, ldns, lhttp, dataDir, domain string
	rtimeout, wtimeout                 time.Duration
	discover                           bool
	secret                             string
	nameserver                         string
)

func init() {
	flag.StringVar(&join, "join", "", "Member of SkyDNS cluster to join can be comma separated list")
	flag.BoolVar(&discover, "discover", false, "Auto discover SkyDNS cluster. Performs an NS lookup on the -domain to find SkyDNS members")
	flag.StringVar(&domain, "domain",
		func() string {
			if x := os.Getenv("SKYDNS_DOMAIN"); x != "" {
				return x
			}
			return "skydns.local"
		}(), "Domain to anchor requests to or env. var. SKYDNS_DOMAIN")
	flag.StringVar(&ldns, "dns",
		func() string {
			if x := os.Getenv("SKYDNS_DNS"); x != "" {
				return x
			}
			return "127.0.0.1:53"
		}(), "IP:Port to bind to for DNS or env. var SKYDNS_DNS")
	flag.StringVar(&lhttp, "http",
		func() string {
			if x := os.Getenv("SKYDNS"); x != "" {
				// get rid of http or https
				x1 := strings.TrimPrefix(x, "https://")
				x1 = strings.TrimPrefix(x1, "http://")
				return x1
			}
			return "127.0.0.1:8080"
		}(), "IP:Port to bind to for HTTP or env. var. SKYDNS")
	flag.StringVar(&dataDir, "data", "./data", "SkyDNS data directory")
	flag.DurationVar(&rtimeout, "rtimeout", 2*time.Second, "Read timeout")
	flag.DurationVar(&wtimeout, "wtimeout", 2*time.Second, "Write timeout")
	flag.StringVar(&secret, "secret", "", "Shared secret for use with http api")
	flag.StringVar(&nameserver, "nameserver", "", "Nameserver address to forward (non-local) queries to e.g. 8.8.8.8:53,8.8.4.4:53")
}

func main() {
	members := make([]string, 0)
	raft.SetLogLevel(0)
	flag.Parse()
	nameservers := strings.Split(nameserver, ",")
	// empty argument given
	if len(nameservers) == 1 && nameservers[0] == "" {
		nameservers = make([]string, 0)
		config, err := dns.ClientConfigFromFile("/etc/resolv.conf")
		if err == nil {
			for _, s := range config.Servers {
				nameservers = append(nameservers, net.JoinHostPort(s, config.Port))
			}
		} else {
			log.Fatal(err)
			return
		}
	}

	if discover {
		ns, err := net.LookupNS(domain)

		if err != nil {
			log.Fatal(err)
			return
		}

		if len(ns) < 1 {
			log.Fatal("No NS records found for ", domain)
			return
		}

		for _, n := range ns {
			members = append(members, strings.TrimPrefix(n.Host, "."))
		}
	} else if join != "" {
		members = strings.Split(join, ",")
	}

	s := server.NewServer(members, domain, ldns, lhttp, dataDir, rtimeout, wtimeout, secret, nameservers)

	stats.Collect()

	waiter, err := s.Start()
	if err != nil {
		log.Fatal(err)
		return
	}
	waiter.Wait()
}
