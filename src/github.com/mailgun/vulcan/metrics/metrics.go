package metrics

import (
	"fmt"
	gmetrics "github.com/rcrowley/go-metrics"
	"net"
	"sync"
)

type HttpMetrics struct {
	prefix         string
	rcmutex        sync.Mutex
	ResponseCodes  map[int]gmetrics.Meter
	ResponseRanges map[string]gmetrics.Meter
}

type ProxyMetrics struct {
	Requests        gmetrics.Meter
	CmdReply        gmetrics.Meter
	CmdForward      gmetrics.Meter
	RequestBodySize gmetrics.Histogram
}

type UpstreamMetrics struct {
	Requests  gmetrics.Meter
	Failovers gmetrics.Meter
	Latency   gmetrics.Timer
	Http      *HttpMetrics
}

func NewHttpMetrics(prefix string) HttpMetrics {
	hm := HttpMetrics{prefix: prefix,
		ResponseRanges: map[string]gmetrics.Meter{
			"1XX": gmetrics.NewMeter(),
			"2XX": gmetrics.NewMeter(),
			"3XX": gmetrics.NewMeter(),
			"4XX": gmetrics.NewMeter(),
			"5XX": gmetrics.NewMeter(),
			"UNK": gmetrics.NewMeter(),
		},
		ResponseCodes: make(map[int]gmetrics.Meter),
	}

	for k, v := range hm.ResponseRanges {
		gmetrics.Register(fmt.Sprintf("%s.http.%s", prefix, k), v)
	}

	return hm
}

func NewProxyMetrics() ProxyMetrics {
	pm := ProxyMetrics{
		Requests:        gmetrics.NewMeter(),
		CmdReply:        gmetrics.NewMeter(),
		CmdForward:      gmetrics.NewMeter(),
		RequestBodySize: gmetrics.NewHistogram(gmetrics.NewExpDecaySample(1028, 0.015)),
	}

	gmetrics.Register("vulcan.proxy.requests", pm.Requests)
	gmetrics.Register("vulcan.proxy.cmd_reply", pm.CmdReply)
	gmetrics.Register("vulcan.proxy.cmd_forward", pm.CmdForward)
	gmetrics.Register("vulcan.proxy.request_body_size", pm.RequestBodySize)

	return pm
}

var upstreamsLock sync.Mutex
var upstreams map[string]*UpstreamMetrics

func init() {
	upstreams = make(map[string]*UpstreamMetrics)
}

func GetUpstreamMetrics(upstreamId string) *UpstreamMetrics {
	upstreamsLock.Lock()
	defer upstreamsLock.Unlock()

	if um, ok := upstreams[upstreamId]; ok {
		return um
	}

	um := NewUpstreamMetrics(upstreamId)
	upstreams[upstreamId] = &um
	return &um
}

func NewUpstreamMetrics(upstreamId string) UpstreamMetrics {
	hm := NewHttpMetrics(fmt.Sprintf("vulcan.upstream.%s", upstreamId))
	um := UpstreamMetrics{
		Requests:  gmetrics.NewMeter(),
		Failovers: gmetrics.NewMeter(),
		Latency:   gmetrics.NewTimer(),
		Http:      &hm,
	}

	gmetrics.Register(fmt.Sprintf("vulcan.upstream.%s.requests", upstreamId), um.Requests)
	gmetrics.Register(fmt.Sprintf("vulcan.upstream.%s.latency", upstreamId), um.Failovers)
	gmetrics.Register(fmt.Sprintf("vulcan.upstream.%s.failovers", upstreamId), um.Failovers)

	return um
}

func AddOutput(outputType string) {
	switch outputType {
	// TODO(pquerna): Add graphite/statsd support
	case "graphite":
		addr, _ := net.ResolveTCPAddr("tcp", "127.0.0.1:2003")
		go gmetrics.Graphite(gmetrics.DefaultRegistry, 10e9, "metrics", addr)
	case "console":
		ConsoleOutput()
	default:
		panic("unrecognized output type")
	}
}

func (hm *HttpMetrics) markSingleCode(statusCode int) {
	// TOOD(pquerna): profile to see if this is horrible
	hm.rcmutex.Lock()
	defer hm.rcmutex.Unlock()
	var meter gmetrics.Meter

	meter, ok := hm.ResponseCodes[statusCode]

	if !ok {
		hm.ResponseCodes[statusCode] = gmetrics.NewMeter()
		meter = hm.ResponseCodes[statusCode]
		gmetrics.Register(fmt.Sprintf("%s.http.status.%d", hm.prefix, statusCode), meter)
	}

	meter.Mark(1)
}

func (hm *HttpMetrics) MarkResponseCode(statusCode int) {
	hm.markSingleCode(statusCode)
	if statusCode >= 500 {
		hm.ResponseRanges["5XX"].Mark(1)
	} else if statusCode >= 200 && statusCode <= 299 {
		hm.ResponseRanges["2XX"].Mark(1)
	} else if statusCode >= 400 && statusCode <= 499 {
		hm.ResponseRanges["2XX"].Mark(1)
	} else if statusCode >= 300 && statusCode <= 399 {
		hm.ResponseRanges["3XX"].Mark(1)
	} else if statusCode >= 100 && statusCode <= 199 {
		hm.ResponseRanges["1XX"].Mark(1)
	} else {
		hm.ResponseRanges["UNK"].Mark(1)
	}
}
