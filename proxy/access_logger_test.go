package proxy

import (
	"bytes"
	"github.com/cloudfoundry/gorouter/route"
	"github.com/cloudfoundry/loggregatorlib/logmessage"
	. "launchpad.net/gocheck"
	"net/http"
	"net/url"
	"regexp"
	"runtime"
	"time"
)

type AccessLoggerSuite struct{}

var _ = Suite(&AccessLoggerSuite{})

var logMessageRegex = `` +
	regexp.QuoteMeta(`foo.bar `) +
	regexp.QuoteMeta(`- `) +
	`\[\d{2}/\d{2}/\d{4}:\d{2}:\d{2}:\d{2} [+-]\d{4}\] ` +
	regexp.QuoteMeta(`"GET /quz?wat HTTP/1.1" `) +
	regexp.QuoteMeta(`200 `) +
	regexp.QuoteMeta(`42 `) +
	regexp.QuoteMeta(`"referer" `) +
	regexp.QuoteMeta(`"user-agent" `) +
	regexp.QuoteMeta(`1.2.3.4:5678 `) +
	regexp.QuoteMeta(`response_time:0.200000000 `) +
	regexp.QuoteMeta(`app_id:my_awesome_id`)

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

func (s *AccessLoggerSuite) TestAccessLogRecordEncode(c *C) {
	r := s.CreateAccessLogRecord()

	b := &bytes.Buffer{}
	_, err := r.WriteTo(b)
	c.Assert(err, IsNil)

	c.Check(b.String(), Matches, "^"+logMessageRegex+"\n")
}

type fakeFile struct {
	payload []byte
}

func (f *fakeFile) Write(data []byte) (int, error) {
	f.payload = data
	return 12, nil
}

type mockEmitter struct {
	emitted bool
	appId   string
	message string
}

func (m *mockEmitter) Emit(appid, message string) {
	m.emitted = true
	m.appId = appid
	m.message = message
}

func (m *mockEmitter) EmitLogMessage(l *logmessage.LogMessage) {

}

func (s *AccessLoggerSuite) TestEmittingOfLogRecords(c *C) {
	accessLogger := NewAccessLogger(nil, "localhost:9843", "secret", 42)
	testEmitter := &mockEmitter{emitted: false}
	accessLogger.e = testEmitter

	accessLogger.Log(*s.CreateAccessLogRecord())
	go accessLogger.Run()
	runtime.Gosched()

	c.Check(testEmitter.emitted, Equals, true)
	c.Check(testEmitter.appId, Equals, "my_awesome_id")
	c.Check(testEmitter.message, Matches, "^"+logMessageRegex+"\n")
	accessLogger.Stop()
}

func (s *AccessLoggerSuite) TestNotEmittingLogRecordsWithNoAppId(c *C) {
	accessLogger := NewAccessLogger(nil, "localhost:9843", "secret", 42)
	testEmitter := &mockEmitter{emitted: false}
	accessLogger.e = testEmitter

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
	var fakeFile = new(fakeFile)

	accessLogger := NewAccessLogger(fakeFile, "localhost:9843", "secret", 42)

	accessLogger.Log(*s.CreateAccessLogRecord())
	go accessLogger.Run()
	runtime.Gosched()

	c.Check(string(fakeFile.payload), Matches, "^"+logMessageRegex+"\n")
	accessLogger.Stop()
}

func (s *AccessLoggerSuite) TestRealEmitterIsNotNil(c *C) {
	accessLogger := NewAccessLogger(nil, "localhost:9843", "secret", 42)

	c.Assert(accessLogger.e, Not(IsNil))
}

func (s *AccessLoggerSuite) TestNotCreatingEmitterWhenNoValidUrlIsGiven(c *C) {
	accessLogger := NewAccessLogger(nil, "this_is_not_a_url", "secret", 42)
	c.Assert(accessLogger.e, IsNil)
	accessLogger.Stop()

	accessLogger = NewAccessLogger(nil, "localhost", "secret", 42)
	c.Assert(accessLogger.e, IsNil)
	accessLogger.Stop()

	accessLogger = NewAccessLogger(nil, "10.10.16.14", "secret", 42)
	c.Assert(accessLogger.e, IsNil)
	accessLogger.Stop()

	accessLogger = NewAccessLogger(nil, "", "secret", 42)
	c.Assert(accessLogger.e, IsNil)
	accessLogger.Stop()
}

func (s *AccessLoggerSuite) TestCreatingEmitterWithIPAddressAndPort(c *C) {
	accessLogger := NewAccessLogger(nil, "10.10.16.14:5432", "secret", 42)

	c.Assert(accessLogger.e, NotNil)
	accessLogger.Stop()
}

func (s *AccessLoggerSuite) TestCreatingEmitterWithLocalhostt(c *C) {
	accessLogger := NewAccessLogger(nil, "localhost:123", "secret", 42)

	c.Assert(accessLogger.e, NotNil)
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
