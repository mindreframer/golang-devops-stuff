package access_log

import (
	"github.com/cloudfoundry/gorouter/route"
	"github.com/cloudfoundry/gorouter/test_util"
	"github.com/cloudfoundry/loggregatorlib/logmessage"

	. "launchpad.net/gocheck"

	"net/http"
	"net/url"
	"runtime"
	"time"
)

type AccessLoggerSuite struct{}

var _ = Suite(&AccessLoggerSuite{})

func (s *AccessLoggerSuite) CreateAccessLogRecord() *AccessLogRecord {
	u, err := url.Parse("http://foo.bar:1234/quz?wat")
	if err != nil {
		panic(err)
	}

	req := &http.Request{
		Method:     "GET",
		URL:        u,
		Proto:      "HTTP/1.1",
		Header:     make(http.Header),
		Host:       "foo.bar",
		RemoteAddr: "1.2.3.4:5678",
	}

	req.Header.Set("Referer", "referer")
	req.Header.Set("User-Agent", "user-agent")

	res := &http.Response{
		StatusCode: http.StatusOK,
	}

	b := &route.Endpoint{
		ApplicationId: "my_awesome_id",
		Host:          "127.0.0.1",
		Port:          4567,
	}

	r := AccessLogRecord{
		Request:       req,
		Response:      res,
		RouteEndpoint: b,
		StartedAt:     time.Unix(10, 100000000),
		FirstByteAt:   time.Unix(10, 200000000),
		FinishedAt:    time.Unix(10, 300000000),
		BodyBytesSent: 42,
	}

	return &r
}

type mockEmitter struct {
	emitted bool
	appId   string
	message string
	done    chan bool
}

func (m *mockEmitter) Emit(appid, message string) {
	m.emitted = true
	m.appId = appid
	m.message = message
	m.done <- true
}

func (m *mockEmitter) EmitError(appid, message string) {
}

func (m *mockEmitter) EmitLogMessage(l *logmessage.LogMessage) {
}

func NewMockEmitter() *mockEmitter {
	return &mockEmitter{
		emitted: false,
		done: make(chan bool, 1),
	}
}

func (s *AccessLoggerSuite) TestEmittingOfLogRecords(c *C) {
	accessLogger := NewFileAndLoggregatorAccessLogger(nil, "localhost:9843", "secret", 42)
	testEmitter := NewMockEmitter()
	accessLogger.emitter = testEmitter

	accessLogger.Log(*s.CreateAccessLogRecord())
	go accessLogger.Run()
	runtime.Gosched()

	timeout := make(chan bool, 1)
	go func() {
		time.Sleep(1*time.Second)
		timeout <- true
	}()

	select {
	case <-testEmitter.done:
			c.Check(testEmitter.emitted, Equals, true)
			c.Check(testEmitter.appId, Equals, "my_awesome_id")
			c.Check(testEmitter.message, Matches, "^.*foo.bar.*\n")
	case <-timeout:
			c.FailNow()
	}

	accessLogger.Stop()
}

func (s *AccessLoggerSuite) TestNotEmittingLogRecordsWithNoAppId(c *C) {
	accessLogger := NewFileAndLoggregatorAccessLogger(nil, "localhost:9843", "secret", 42)
	testEmitter := NewMockEmitter()
	accessLogger.emitter = testEmitter

	routeEndpoint := &route.Endpoint{
		ApplicationId: "",
		Host:          "127.0.0.1",
		Port:          4567,
	}

	accessLogRecord := s.CreateAccessLogRecord()
	accessLogRecord.RouteEndpoint = routeEndpoint
	accessLogger.Log(*accessLogRecord)
	go accessLogger.Run()
	runtime.Gosched()

	c.Check(testEmitter.emitted, Equals, false)
	accessLogger.Stop()
}

func (s *AccessLoggerSuite) TestWritingOfLogRecordsToTheFile(c *C) {
	var fakeFile = new(test_util.FakeFile)

	accessLogger := NewFileAndLoggregatorAccessLogger(fakeFile, "localhost:9843", "secret", 42)

	accessLogger.Log(*s.CreateAccessLogRecord())
	go accessLogger.Run()
	runtime.Gosched()

	c.Check(string(fakeFile.Payload), Matches, "^.*foo.bar.*\n")
	accessLogger.Stop()
}

func (s *AccessLoggerSuite) TestRealEmitterIsNotNil(c *C) {
	accessLogger := NewFileAndLoggregatorAccessLogger(nil, "localhost:9843", "secret", 42)

	c.Assert(accessLogger.emitter, Not(IsNil))
}

func (s *AccessLoggerSuite) TestNotCreatingEmitterWhenNoValidUrlIsGiven(c *C) {
	accessLogger := NewFileAndLoggregatorAccessLogger(nil, "this_is_not_a_url", "secret", 42)
	c.Assert(accessLogger.emitter, IsNil)
	accessLogger.Stop()

	accessLogger = NewFileAndLoggregatorAccessLogger(nil, "localhost", "secret", 42)
	c.Assert(accessLogger.emitter, IsNil)
	accessLogger.Stop()

	accessLogger = NewFileAndLoggregatorAccessLogger(nil, "10.10.16.14", "secret", 42)
	c.Assert(accessLogger.emitter, IsNil)
	accessLogger.Stop()

	accessLogger = NewFileAndLoggregatorAccessLogger(nil, "", "secret", 42)
	c.Assert(accessLogger.emitter, IsNil)
	accessLogger.Stop()
}

func (s *AccessLoggerSuite) TestCreatingEmitterWithIPAddressAndPort(c *C) {
	accessLogger := NewFileAndLoggregatorAccessLogger(nil, "10.10.16.14:5432", "secret", 42)

	c.Assert(accessLogger.emitter, NotNil)
	accessLogger.Stop()
}

func (s *AccessLoggerSuite) TestCreatingEmitterWithLocalhostt(c *C) {
	accessLogger := NewFileAndLoggregatorAccessLogger(nil, "localhost:123", "secret", 42)

	c.Assert(accessLogger.emitter, NotNil)
	accessLogger.Stop()
}

type nullWriter struct{}

func (n nullWriter) Write(b []byte) (int, error) {
	return len(b), nil
}

func (s *AccessLoggerSuite) BenchmarkAccessLogRecordWriteTo(c *C) {
	r := s.CreateAccessLogRecord()
	w := nullWriter{}

	for i := 0; i < c.N; i++ {
		r.WriteTo(w)
	}
}
