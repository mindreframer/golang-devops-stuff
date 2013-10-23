package errplane

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ErrplaneCollectorApiSuite struct{}

var _ = Suite(&ErrplaneCollectorApiSuite{})

var (
	recorder    *HttpRequestRecorder
	listener    net.Listener
	currentTime time.Time
)

type HttpRequestRecorder struct {
	requests [][]byte
	forms    []url.Values
}

func (self *HttpRequestRecorder) ServeHTTP(writer http.ResponseWriter, req *http.Request) {
	data, _ := ioutil.ReadAll(req.Body)
	self.requests = append(self.requests, data)
	req.ParseForm()
	self.forms = append(self.forms, req.Form)
	writer.WriteHeader(http.StatusCreated)
}

func (s *ErrplaneCollectorApiSuite) SetUpSuite(c *C) {
	var err error
	listener, err = net.Listen("tcp4", "")
	c.Assert(err, IsNil)
	recorder = new(HttpRequestRecorder)
	http.Handle("/databases/app4you2lovestaging/points", recorder)
	go func() { http.Serve(listener, nil) }()

	currentTime = time.Now()
}

func (s *ErrplaneCollectorApiSuite) SetUpTest(c *C) {
	recorder.requests = nil
	recorder.forms = nil
}

func (s *ErrplaneCollectorApiSuite) TearDownSuite(c *C) {
	listener.Close()
}

func (s *ErrplaneCollectorApiSuite) TestApi(c *C) {
	ep := newTestClient("app4you2love", "staging", "some_key")
	c.Assert(ep, NotNil)
	ep.SetHttpHost(listener.Addr().(*net.TCPAddr).String())

	ep.Report("some_metric", 123.4, currentTime, "some_context", Dimensions{
		"foo": "bar",
	})

	ep.Close() // make sure we flush all the points

	c.Assert(recorder.requests, HasLen, 1)
	expected := fmt.Sprintf(
		`[{"n":"some_metric","p":[{"v":123.4,"c":"some_context","t":%d,"d":{"foo":"bar"}}]}]`,
		currentTime.UnixNano()/int64(time.Second))
	c.Assert(string(recorder.requests[0]), Equals, expected)
	c.Assert(recorder.forms, HasLen, 1)
	c.Assert(recorder.forms[0].Get("api_key"), Equals, "some_key")
}

func (s *ErrplaneCollectorApiSuite) TestApiHeartbeat(c *C) {
	ep := newTestClient("app4you2love", "staging", "some_key")
	c.Assert(ep, NotNil)
	ep.SetHttpHost(listener.Addr().(*net.TCPAddr).String())

	ep.Heartbeat("heartbeat_metric", time.Second, "", nil)
	time.Sleep(1 * time.Second)
	ep.Close() // make sure we flush all the points

	c.Assert(recorder.requests, HasLen, 1)
	epocTime := currentTime.UnixNano() / int64(time.Second)
	data := make([]*JsonPoints, 0)
	err := json.Unmarshal(recorder.requests[0], &data)
	c.Assert(err, IsNil)
	c.Assert(data, HasLen, 1)
	c.Assert(data[0].Name, Equals, "heartbeat_metric")
	points := data[0].Points
	c.Assert(points, HasLen, 1)
	if points[0].Time < epocTime {
		epocTime -= 1
	} else if points[0].Time > epocTime {
		epocTime += 1
	}
	c.Assert(points[0].Time, Equals, epocTime)
	c.Assert(recorder.forms, HasLen, 1)
	c.Assert(recorder.forms[0].Get("api_key"), Equals, "some_key")
}

func (s *ErrplaneCollectorApiSuite) TestApiAggregatesPoints(c *C) {
	ep := newTestClient("app4you2love", "staging", "some_key")
	c.Assert(ep, NotNil)
	ep.SetHttpHost(listener.Addr().(*net.TCPAddr).String())

	ep.Report("some_metric", 123.4, currentTime, "some_context", Dimensions{
		"foo": "bar",
	})

	ep.Report("some_metric", 567.8, currentTime, "different_context", Dimensions{
		"foo": "bar",
	})

	ep.Report("different_metric", 123.4, currentTime, "some_context", Dimensions{
		"foo": "bar",
	})

	ep.Close() // make sure we flush all the points

	c.Assert(recorder.requests, HasLen, 1)
	epocTime := currentTime.UnixNano() / int64(time.Second)
	expected := fmt.Sprintf(
		`[{"n":"some_metric","p":[{"v":123.4,"c":"some_context","t":%d,"d":{"foo":"bar"}},{"v":567.8,"c":"different_context","t":%d,"d":{"foo":"bar"}}]},{"n":"different_metric","p":[{"v":123.4,"c":"some_context","t":%d,"d":{"foo":"bar"}}]}]`, epocTime, epocTime, epocTime)
	c.Assert(string(recorder.requests[0]), Equals, expected)
	c.Assert(recorder.forms, HasLen, 1)
	c.Assert(recorder.forms[0].Get("api_key"), Equals, "some_key")
}

func (s *ErrplaneCollectorApiSuite) TestApiRejectInvalidNames(c *C) {
	ep := newTestClient("app4you2love", "staging", "some_key")
	c.Assert(ep, NotNil)
	c.Assert(ep.Report("invalid/metric/name", 1.0, time.Now(), "", nil), NotNil)
}

func (s *ErrplaneCollectorApiSuite) TestApiRejectLongMetricNames(c *C) {
	metricName := strings.Repeat("long_metric", 100)
	ep := newTestClient("app4you2love", "staging", "some_key")
	c.Assert(ep, NotNil)
	c.Assert(ep.Report(metricName, 1.0, time.Now(), "", nil), NotNil)
}
