package errplane

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"runtime"
	"time"
)

type WriteOperation struct {
	Database  string        `json:"d"`
	ApiKey    string        `json:"a"`
	Operation string        `json:"o,omitempty"`
	Writes    []*JsonPoints `json:"w"`
}

type JsonPoint struct {
	Value      float64           `json:"v"`
	Context    string            `json:"c,omitempty"`
	Time       int64             `json:"t,omitempty"`
	Dimensions map[string]string `json:"d,omitempty"`
}

type JsonPoints struct {
	Name   string       `json:"n"`
	Points []*JsonPoint `json:"p"`
}

type Dimensions map[string]string

type PostType int

const (
	UDP PostType = iota
	HTTP
)

var METRIC_REGEX, _ = regexp.Compile("^[a-zA-Z0-9._]*$")

type ErrplanePost struct {
	postType  PostType
	operation *WriteOperation
}

type Errplane struct {
	proto               string
	udpConn             *net.UDPConn
	url                 string
	apiKey              string
	database            string
	Timeout             time.Duration
	closeChan           chan bool
	msgChan             chan *ErrplanePost
	closed              bool
	timeout             time.Duration
	runtimeStatsRunning bool
}

const (
	DEFAULT_HTTP_HOST = "w.apiv3.errplane.com"
	DEFAULT_UDP_ADDR  = "udp.apiv3.errplane.com:8126"
)

// Initializer.
//   app: the application key from the Settings/Applications page
//   environment: the environment from the Settings/Applications page
//   apiKey: the api key from Settings/Orginzations page
func New(app, environment, apiKey string) *Errplane {
	return newCommon("https", app, environment, apiKey)
}

func newTestClient(app, environment, apiKey string) *Errplane {
	return newCommon("http", app, environment, apiKey)
}

func newCommon(proto, app, environment, apiKey string) *Errplane {
	database := fmt.Sprintf("%s%s", app, environment)
	ep := &Errplane{
		proto:     proto,
		database:  database,
		apiKey:    apiKey,
		Timeout:   1 * time.Second,
		msgChan:   make(chan *ErrplanePost),
		closeChan: make(chan bool),
		closed:    false,
		timeout:   2 * time.Second,
	}
	ep.SetHttpHost(DEFAULT_HTTP_HOST)
	ep.SetUdpAddr(DEFAULT_UDP_ADDR)
	ep.setTransporter(nil)
	go ep.processMessages()
	return ep
}

// call from a goroutine, this method never returns
func (self *Errplane) processMessages() {
	posts := make([]*ErrplanePost, 0)
	for {

		select {
		case x := <-self.msgChan:
			posts = append(posts, x)
			if len(posts) < 100 {
				continue
			}
			self.flushPosts(posts)
		case <-time.After(1 * time.Second):
			self.flushPosts(posts)
		case <-self.closeChan:
			self.flushPosts(posts)
			self.closeChan <- true
			return
		}

		posts = make([]*ErrplanePost, 0)
	}
}

func (self *Errplane) flushPosts(posts []*ErrplanePost) {
	if len(posts) == 0 {
		return
	}

	var (
		httpPoints         = make([]*WriteOperation, 0)
		udpReportPoints    = make([]*WriteOperation, 0)
		udpSumPoints       = make([]*WriteOperation, 0)
		udpAggregatePoints = make([]*WriteOperation, 0)
	)

	for _, post := range posts {
		operation := post.operation
		if post.postType == UDP {
			switch operation.Operation {
			case "r":
				udpReportPoints = append(udpReportPoints, operation)
			case "t":
				udpAggregatePoints = append(udpAggregatePoints, operation)
			case "c":
				udpSumPoints = append(udpSumPoints, operation)
			default:
				panic(fmt.Errorf("Unknown point type %s", operation.Operation))
			}
		} else {
			httpPoints = append(httpPoints, operation)
		}
	}

	// do the http ones first
	httpPoint := self.mergeMetrics(httpPoints)
	if httpPoint != nil {
		if err := self.SendHttp(httpPoint); err != nil {
			fmt.Fprintf(os.Stderr, "Error while posting points to Errplane. Error: %s\n", err)
		}
	}

	// do the udp points here
	udpReportPoint := self.mergeMetrics(udpReportPoints)
	if udpReportPoint != nil {
		udpReportPoint.Operation = "r"
		if err := self.SendUdp(udpReportPoint); err != nil {
			fmt.Fprintf(os.Stderr, "Error while posting points to Errplane. Error: %s\n", err)
		}
	}
	udpAggregatePoint := self.mergeMetrics(udpAggregatePoints)
	if udpAggregatePoint != nil {
		udpAggregatePoint.Operation = "t"
		if err := self.SendUdp(udpAggregatePoint); err != nil {
			fmt.Fprintf(os.Stderr, "Error while posting points to Errplane. Error: %s\n", err)
		}
	}
	udpSumPoint := self.mergeMetrics(udpSumPoints)
	if udpSumPoint != nil {
		udpSumPoint.Operation = "c"
		if err := self.SendUdp(udpSumPoint); err != nil {
			fmt.Fprintf(os.Stderr, "Error while posting points to Errplane. Error: %s\n", err)
		}
	}
}

func (self *Errplane) Heartbeat(name string, interval time.Duration, context string, dimensions Dimensions) {
	go func() {
		for {
			if self.closed {
				return
			}

			self.Report(name, 1.0, time.Now(), context, dimensions)
			time.Sleep(interval)
		}
	}()
}

func (self *Errplane) SendHttp(data *WriteOperation) error {
	buf, err := json.Marshal(data.Writes)
	if err != nil {
		return fmt.Errorf("Cannot marshal %#v. Error: %s", data, err)
	}

	resp, err := http.Post(self.url, "application/json", bytes.NewReader(buf))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 201 {
		return fmt.Errorf("Server returned status code %d", resp.StatusCode)
	}
	return nil
}

func (self *Errplane) SendUdp(data *WriteOperation) error {
	buf, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("Cannot marshal %#v. Error: %s", data, err)
	}

	_, err = self.udpConn.Write(buf)
	return err
}

func (self *Errplane) mergeMetrics(operations []*WriteOperation) *WriteOperation {
	if len(operations) == 0 {
		return nil
	}

	metricToPoints := make(map[string][]*JsonPoint)

	for _, operation := range operations {
		for _, jsonPoints := range operation.Writes {
			name := jsonPoints.Name
			metricToPoints[name] = append(metricToPoints[name], jsonPoints.Points...)
		}
	}

	mergedMetrics := make([]*JsonPoints, 0)

	for metric, points := range metricToPoints {
		mergedMetrics = append(mergedMetrics, &JsonPoints{
			Name:   metric,
			Points: points,
		})
	}

	return &WriteOperation{
		Database: self.database,
		ApiKey:   self.apiKey,
		Writes:   mergedMetrics,
	}
}

// Close the errplane object and flush all buffered data points
func (self *Errplane) Close() {
	self.closed = true
	// tell the go routine to finish
	self.closeChan <- true
	// wait for the go routine to finish
	<-self.closeChan
}

func (self *Errplane) SetUdpAddr(addr string) error {
	localAddr, err := net.ResolveUDPAddr("udp4", "")
	if err != nil {
		return err
	}
	remoteAddr, err := net.ResolveUDPAddr("udp4", addr)
	if err != nil {
		return err
	}
	udpConn, err := net.DialUDP("udp4", localAddr, remoteAddr)
	if err != nil {
		return err
	}
	if self.udpConn != nil {
		self.udpConn.Close()
	}
	self.udpConn = udpConn
	return nil
}

func (self *Errplane) SetHttpHost(host string) {
	params := url.Values{}
	params.Set("api_key", self.apiKey)
	self.url = fmt.Sprintf("%s://%s/databases/%s/points?%s", self.proto, host, self.database, params.Encode())
}

func (self *Errplane) SetProxy(proxy string) error {
	proxyUrl, err := url.Parse(proxy)
	if err != nil {
		return err
	}
	self.setTransporter(proxyUrl)
	return nil
}

func (self *Errplane) SetTimeout(timeout time.Duration) error {
	self.timeout = timeout
	self.setTransporter(nil)
	return nil
}

func (self *Errplane) setTransporter(proxyUrl *url.URL) {
	transporter := &http.Transport{}
	if proxyUrl != nil {
		transporter.Proxy = http.ProxyURL(proxyUrl)
	}
	transporter.Dial = func(network, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(network, addr, self.timeout)
		if err != nil {
			return nil, err
		}
		conn.SetDeadline(time.Now().Add(self.timeout))
		return conn, nil
	}
	http.DefaultTransport = transporter
}

// Start a goroutine that will post runtime stats to errplane, stats include memory usage, garbage collection, number of goroutines, etc.
// Args:
//   prefix: the prefix to use in the metric name
//   context: all points will be reported with the given context name
//   dimensions: all points will be reported with the given dimensions
//   sleep: the sampling frequency
func (self *Errplane) ReportRuntimeStats(prefix, context string, dimensions Dimensions, sleep time.Duration) {
	if self.runtimeStatsRunning {
		fmt.Fprintf(os.Stderr, "Runtime stats is already running\n")
		return
	}

	self.runtimeStatsRunning = true
	go self.reportRuntimeStats(prefix, context, dimensions, sleep)
}

func (self *Errplane) StopRuntimeStatsReporting(prefix, context string, dimensions Dimensions, sleep time.Duration) {
	self.runtimeStatsRunning = false
}

func (self *Errplane) reportRuntimeStats(prefix, context string, dimensions Dimensions, sleep time.Duration) {
	memStats := &runtime.MemStats{}
	lastSampleTime := time.Now()
	var lastPauseNs uint64 = 0
	var lastNumGc uint32 = 0

	nsInMs := float64(time.Millisecond)

	for self.runtimeStatsRunning {
		runtime.ReadMemStats(memStats)

		now := time.Now()

		self.Report(fmt.Sprintf("%s.goroutines", prefix), float64(runtime.NumGoroutine()), now, context, dimensions)
		self.Report(fmt.Sprintf("%s.memory.heap.objects", prefix), float64(memStats.HeapObjects), now, context, dimensions)
		self.Report(fmt.Sprintf("%s.memory.allocated", prefix), float64(memStats.Alloc), now, context, dimensions)
		self.Report(fmt.Sprintf("%s.memory.mallocs", prefix), float64(memStats.Mallocs), now, context, dimensions)
		self.Report(fmt.Sprintf("%s.memory.frees", prefix), float64(memStats.Frees), now, context, dimensions)
		self.Report(fmt.Sprintf("%s.memory.gc.total_pause", prefix), float64(memStats.PauseTotalNs)/nsInMs, now, context, dimensions)
		self.Report(fmt.Sprintf("%s.memory.heap", prefix), float64(memStats.HeapAlloc), now, context, dimensions)
		self.Report(fmt.Sprintf("%s.memory.stack", prefix), float64(memStats.StackInuse), now, context, dimensions)

		if lastPauseNs > 0 {
			pauseSinceLastSample := memStats.PauseTotalNs - lastPauseNs
			self.Report(fmt.Sprintf("%s.memory.gc.pause_per_second", prefix), float64(pauseSinceLastSample)/nsInMs/sleep.Seconds(), now, context, dimensions)
		}
		lastPauseNs = memStats.PauseTotalNs

		countGc := int(memStats.NumGC - lastNumGc)
		if lastNumGc > 0 {
			diff := float64(countGc)
			diffTime := now.Sub(lastSampleTime).Seconds()
			self.Report(fmt.Sprintf("%s.memory.gc.gc_per_second", prefix), diff/diffTime, now, context, dimensions)
		}

		// get the individual pause times
		if countGc > 0 {
			if countGc > 256 {
				fmt.Fprintf(os.Stderr, "We're missing some gc pause times")
				countGc = 256
			}

			for i := 0; i < countGc; i++ {
				idx := int((memStats.NumGC-uint32(i))+255) % 256
				pause := float64(memStats.PauseNs[idx])
				self.Aggregate(fmt.Sprintf("%s.memory.gc.pause", prefix), pause/nsInMs, context, dimensions)
			}
		}

		// keep track of the previous state
		lastNumGc = memStats.NumGC
		lastSampleTime = now

		time.Sleep(sleep)
	}
}

// FIXME: make timestamp, context and dimensions optional (accept empty values, e.g. nil)
func (self *Errplane) Report(metric string, value float64, timestamp time.Time, context string, dimensions Dimensions) error {
	return self.sendCommon("", metric, value, &timestamp, context, dimensions, HTTP)
}

func (self *Errplane) sendUdpPayload(metricType, metric string, value float64, context string, dimensions Dimensions) error {
	return self.sendCommon(metricType, metric, value, nil, context, dimensions, UDP)
}

func (self *Errplane) sendCommon(metricType, metric string, value float64, timestamp *time.Time, context string, dimensions Dimensions, postType PostType) error {
	if err := verifyMetricName(metric); err != nil {
		return err
	}
	point := &JsonPoint{
		Value:      value,
		Context:    context,
		Dimensions: dimensions,
	}

	if timestamp != nil {
		point.Time = timestamp.Unix()
	}

	data := &WriteOperation{
		Operation: metricType,
		Writes: []*JsonPoints{
			&JsonPoints{
				Name:   metric,
				Points: []*JsonPoint{point},
			},
		},
	}
	self.msgChan <- &ErrplanePost{postType, data}
	return nil
}

func (self *Errplane) ReportUDP(metric string, value float64, context string, dimensions Dimensions) error {
	return self.sendUdpPayload("r", metric, value, context, dimensions)
}

func (self *Errplane) Aggregate(metric string, value float64, context string, dimensions Dimensions) error {
	return self.sendUdpPayload("t", metric, value, context, dimensions)
}

func (self *Errplane) Sum(metric string, value float64, context string, dimensions Dimensions) error {
	return self.sendUdpPayload("c", metric, float64(value), context, dimensions)
}

func verifyMetricName(name string) error {
	if len(name) > 255 {
		return fmt.Errorf("Metric names must be less than 255 characters")
	}

	if !METRIC_REGEX.MatchString(name) {
		return fmt.Errorf("Invalid metric name %s. See docs for valid metric names", name)
	}

	return nil
}
