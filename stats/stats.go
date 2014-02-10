// Copyright (c) 2013 The SkyDNS Authors. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package stats

import (
	"flag"
	"github.com/rcrowley/go-metrics"
	"github.com/rcrowley/go-metrics/influxdb"
	"github.com/rcrowley/go-metrics/stathat"
	"log"
	"net"
	"os"
)

var (
	ExpiredCount       metrics.Counter
	RequestCount       metrics.Counter
	AddServiceCount    metrics.Counter
	UpdateTTLCount     metrics.Counter
	GetServiceCount    metrics.Counter
	RemoveServiceCount metrics.Counter

	metricsToStdErr             bool
	graphiteServer, stathatUser string
	influxConfig                *influxdb.Config
)

func init() {
	influxConfig = &influxdb.Config{}

	flag.BoolVar(&metricsToStdErr, "metricsToStdErr", false, "Write metrics to stderr periodically")
	flag.StringVar(&graphiteServer, "graphiteServer", "", "Graphite Server connection string e.g. 127.0.0.1:2003")
	flag.StringVar(&stathatUser, "stathatUser", "", "StatHat account for metrics")
	flag.StringVar(&influxConfig.Host, "influxdbHost", "", "Influxdb host address for metrics")
	flag.StringVar(&influxConfig.Database, "influxdbDatabase", "", "Influxdb database name for metrics")
	flag.StringVar(&influxConfig.Username, "influxdbUsername", "", "Influxdb username for metrics")
	flag.StringVar(&influxConfig.Password, "influxdbPassword", "", "Influxdb password for metrics")

	ExpiredCount = metrics.NewCounter()
	metrics.Register("skydns-expired-entries", ExpiredCount)

	RequestCount = metrics.NewCounter()
	metrics.Register("skydns-requests", RequestCount)

	AddServiceCount = metrics.NewCounter()
	metrics.Register("skydns-add-service-requests", AddServiceCount)

	UpdateTTLCount = metrics.NewCounter()
	metrics.Register("skydns-update-ttl-requests", UpdateTTLCount)

	GetServiceCount = metrics.NewCounter()
	metrics.Register("skydns-get-service-requests", GetServiceCount)

	RemoveServiceCount = metrics.NewCounter()
	metrics.Register("skydns-remove-service-requests", RemoveServiceCount)
}

func Collect() {
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

	if influxConfig.Host != "" {
		go influxdb.Influxdb(metrics.DefaultRegistry, 10e9, influxConfig)
	}
}
