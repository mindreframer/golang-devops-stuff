// Copyright (c) 2013 Erik St. Martin, Brian Ketelsen. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package main

import (
	"flag"
	"github.com/goraft/raft"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/stathat"
	"github.com/skynetservices/skydns/server"
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
	metricsToStdErr                    bool
	graphiteServer, stathatUser        string
	secret                             string
)

func init() {
	flag.StringVar(&join, "join", "", "Member of SkyDNS cluster to join can be comma separated list")
	flag.BoolVar(&discover, "discover", false, "Auto discover SkyDNS cluster. Performs an NS lookup on the -domain to find SkyDNS members")
	flag.StringVar(&domain, "domain", "skydns.local", "Domain to anchor requests to")
	flag.StringVar(&ldns, "dns", "127.0.0.1:53", "IP:Port to bind to for DNS")
	flag.StringVar(&lhttp, "http", "127.0.0.1:8080", "IP:Port to bind to for HTTP")
	flag.StringVar(&dataDir, "data", "./data", "SkyDNS data directory")
	flag.DurationVar(&rtimeout, "rtimeout", 2*time.Second, "Read timeout")
	flag.DurationVar(&wtimeout, "wtimeout", 2*time.Second, "Write timeout")
	flag.BoolVar(&metricsToStdErr, "metricsToStdErr", false, "Write metrics to stderr periodically")
	flag.StringVar(&graphiteServer, "graphiteServer", "", "Graphite Server connection string e.g. 127.0.0.1:2003")
	flag.StringVar(&stathatUser, "stathatUser", "", "StatHat account for metrics")
	flag.StringVar(&secret, "secret", "", "Shared secret for use with http api")
}

func main() {
	members := make([]string, 0)

	raft.SetLogLevel(0)

	flag.Parse()

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

	s := server.NewServer(members, domain, ldns, lhttp, dataDir, rtimeout, wtimeout, secret)

	// Set up metrics if specified on the command line
	if metricsToStdErr {
		go metrics.Log(metrics.DefaultRegistry, 60e9, log.New(os.Stderr, "metrics: ", log.Lmicroseconds))
	}

	if len(graphiteServer) > 1 {
		addr, err := net.ResolveTCPAddr("tcp", graphiteServer)
		if err != nil {
			go metrics.Graphite(metrics.DefaultRegistry, 10e9, "skydns", addr)
		}
	}

	if len(stathatUser) > 1 {
		go stathat.Stathat(metrics.DefaultRegistry, 10e9, stathatUser)
	}

	waiter, err := s.Start()

	if err != nil {
		log.Fatal(err)
		return
	}

	waiter.Wait()
}
