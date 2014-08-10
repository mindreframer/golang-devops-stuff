// Package fsd is a client library for accessing StatsD daemon. You can specify
// custom StatsD port via statsd flag
package fsd

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"strconv"
	"time"

	"github.com/coreos/go-etcd/etcd"
	"github.com/op/go-logging"
)

var (
	address string

	addressConfig chan *etcd.Response
	outgoing      = make(chan []byte, 100000)

	log = logging.MustGetLogger("fsd")

	conn net.Conn
)

func InitWithDynamicConfig(client *etcd.Client, hostname string) {
	addressConfig = make(chan *etcd.Response)

	go watchConfiguration(client, fmt.Sprintf("%v/statsd/address", hostname))
	go processOutgoing()
}

func getCurrentVersion(client *etcd.Client, key string) *etcd.Response {
	for {
		if resp, err := client.Get(key, false, false); err == nil {
			// failed to fetch first value
			return resp
		} else {
			time.Sleep(time.Second)
		}
	}
}
func watchForUpdates(client *etcd.Client, key string, index uint64) {

	for {
		if _, err := client.Watch(key, index, false, addressConfig, nil); err != nil {
			toSleep := 5 * time.Second

			log.Debug("error watching etcd for key %v: %v", key, err)
			log.Debug("retry in %v", toSleep)
			time.Sleep(toSleep)
		}
	}

}

func watchConfiguration(client *etcd.Client, key string) {

	resp := getCurrentVersion(client, key)
	addressConfig <- resp

	watchForUpdates(client, key, resp.EtcdIndex)
}

func connect() (err error) {
	if address == "" {
		return
	}

	log.Debug("fsd connects to %v", address)
	if conn, err = net.Dial("udp", address); err != nil {
		return
	}

	return
}

func processOutgoing() {
	for {
		select {
		case outgoing := <-outgoing:
			if !hasAddress() {
				break
			}

			if _, err := conn.Write(outgoing); err != nil {
				connect()
			}
		case response := <-addressConfig:
			if response.Node != nil && response.Node.Value != "" {
				address = response.Node.Value
				connect()
			}
		}
	}
}

func hasAddress() bool {
	return address != ""
}

// To read about the different semantics check out
// https://github.com/b/statsd_spec
// http://docs.datadoghq.com/guides/dogstatsd/

// Increment the page.views counter.
// page.views:1|c
func Count(name string, value float64) {
	CountL(name, value, 1.0)
}

func CountL(name string, value float64, rate float64) {
	if rand.Float64() > rate {
		return
	}

	payload, err := rateCheck(rate, createPayload(name, value, "c"))
	if err != nil {
		return
	}

	send(payload)
}

// Record the fuel tank is half-empty
// fuel.level:0.5|g
func Gauge(name string, value float64) {
	payload := createPayload(name, value, "g")
	send(payload)
}

// A request latency
// request.latency:320|ms
// Or a payload of a image
// image.size:2.3|ms
func Timer(name string, duration time.Duration) {
	TimerL(name, duration, 1.0)
}

func TimerL(name string, duration time.Duration, rate float64) {
	if rand.Float64() > rate {
		return
	}

	HistogramL(name, float64(duration.Nanoseconds()/1000000), rate)
}

func Histogram(name string, value float64) {
	HistogramL(name, value, 1.0)
}

func HistogramL(name string, value float64, rate float64) {
	if rand.Float64() > rate {
		return
	}

	payload, err := rateCheck(rate, createPayload(name, value, "ms"))
	if err != nil {
		return
	}

	send(payload)
}

// TimeSince records a named timer with the duration since start
func TimeSince(name string, start time.Time) {
	TimeSinceL(name, start, 1.0)
}

// TimeSince records a rated and named timer with the duration since start
func TimeSinceL(name string, start time.Time, rate float64) {
	if rand.Float64() > rate {
		return
	}

	TimerL(name, time.Now().Sub(start), rate)
}

func Time(name string, lambda func()) {
	TimeL(name, 1.0, lambda)
}

func TimeL(name string, rate float64, lambda func()) {
	if rand.Float64() > rate {
		lambda()
	} else {
		start := time.Now()
		lambda()
		TimeSinceL(name, start, rate)
	}
}

// Track a unique visitor id to the site.
// users.uniques:1234|s
func Set(name string, value float64) {
	payload := createPayload(name, value, "s")
	send(payload)
}

func createPayload(name string, value float64, suffix string) string {
	// we spend a lot of time in this code
	return name + ":" + strconv.FormatFloat(value, 'f', -1, 64) + "|" + suffix
	//return fmt.Sprintf("%s:%f|%s", name, value, suffix)
}

func rateCheck(rate float64, payload string) (string, error) {
	if rate < 1 {
		return payload + fmt.Sprintf("|@%f", rate), nil
	} else { // rate is 1.0 == all samples should be sent
		return payload, nil
	}

	return "", errors.New("Out of rate limit")
}

func send(payload string) {
	length := float64(len(outgoing))
	capacity := float64(cap(outgoing))

	if length < capacity*0.9 {
		outgoing <- []byte(payload)
	}
}
