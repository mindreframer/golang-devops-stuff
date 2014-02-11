package fsd

import (
	"errors"
	"fmt"
	"math/rand"
	"net"
	"time"
)

var (
	Instance *Fsd
)

type Fsd struct {
	outgoing chan string
	address  string
	conn     net.Conn
}

func init() {
	Start("127.0.0.1:8125")
}

func Start(address string) {
	Instance = &Fsd{address: address, outgoing: make(chan string, 100000)}
	Instance.connect()

	go Instance.processOutgoing()
}

func (fsd *Fsd) connect() error {
	conn, err := net.Dial("udp", fsd.address)
	if err != nil {
		return err
	}

	fsd.conn = conn
	return nil
}

func (fsd *Fsd) processOutgoing() {
	for outgoing := range fsd.outgoing {
		data := fmt.Sprintf("%s", outgoing)

		if _, err := fsd.conn.Write([]byte(data)); err != nil {
			Count("fsd.sending.failure", 1)
			fsd.connect()
		} else {
			CountL("fsd.sending.sucess", 1, 0.1)
		}
	}
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
	payload := createPayload(name, value) + "|c"

	suffix, err := rateCheck(rate)
	if err != nil {
		return
	}

	payload = payload + suffix
	send(payload)
}

// Record the fuel tank is half-empty
// fuel.level:0.5|g
func Gauge(name string, value float64) {
	payload := createPayload(name, value) + "|g"
	send(payload)
}

// A request latency
// request.latency:320|ms
// Or a payload of a image
// image.size:2.3|ms
func Timer(name string, value float64) {
	TimerL(name, value, 1.0)
}

func TimerL(name string, value float64, rate float64) {
	payload := createPayload(name, value) + "|ms"

	suffix, err := rateCheck(rate)
	if err != nil {
		return
	}

	payload = payload + suffix
	send(payload)
}

func Time(name string, lambda func()) {
	TimeL(name, 1.0, lambda)
}

func TimeL(name string, rate float64, lambda func()) {
	start := time.Now()
	lambda()
	TimerL(name, float64(time.Now().Sub(start).Nanoseconds()/1000000), rate)
}

// Track a unique visitor id to the site.
// users.uniques:1234|s
func Set(name string, value float64) {
	payload := createPayload(name, value) + "|s"
	send(payload)
}

func createPayload(name string, value float64) (payload string) {
	payload = fmt.Sprintf("%s:%f", name, value)
	return payload
}

func rateCheck(rate float64) (suffix string, err error) {
	if rate < 1 {
		if rand.Float64() < rate {
			return fmt.Sprintf("|@%f", rate), nil
		}
	} else {
		return "", nil
	}

	return "", errors.New("Out of rate limit")
}

func send(payload string) {
	length := float64(len(Instance.outgoing))
	capacity := float64(cap(Instance.outgoing))

	Gauge("fsd.buffer.fillrate", capacity/length)

	if length < capacity*0.9 {
		Instance.outgoing <- payload
		CountL("fsd.buffer.success", 1, 0.1)
	} else {
		CountL("fsd.buffer.failure", 1, 0.1)
	}
}
