package access_log_test

import (
	. "github.com/cloudfoundry/gorouter/access_log"
	"github.com/cloudfoundry/gorouter/route"
	"github.com/cloudfoundry/gorouter/test_util"
	"github.com/cloudfoundry/loggregatorlib/logmessage"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net/http"
	"net/url"
	"time"
)

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
		done:    make(chan bool, 1),
	}
}

var _ = Describe("AccessLog", func() {

	Context("with an emitter", func() {
		It("a record is written", func() {
			testEmitter := NewMockEmitter()
			accessLogger := NewFileAndLoggregatorAccessLogger(nil, testEmitter)
			go accessLogger.Run()

			accessLogger.Log(*CreateAccessLogRecord())
			Eventually(testEmitter.done).Should(Receive())
			Ω(testEmitter.emitted).Should(BeTrue())
			Ω(testEmitter.appId).To(Equal("my_awesome_id"))
			Ω(testEmitter.message).To(MatchRegexp("^.*foo.bar.*\n"))

			accessLogger.Stop()
		})

		It("a record with no app id is not written", func() {
			testEmitter := NewMockEmitter()
			accessLogger := NewFileAndLoggregatorAccessLogger(nil, testEmitter)

			routeEndpoint := route.NewEndpoint("", "127.0.0.1", 4567, "", nil)

			accessLogRecord := CreateAccessLogRecord()
			accessLogRecord.RouteEndpoint = routeEndpoint
			accessLogger.Log(*accessLogRecord)
			go accessLogger.Run()

			Consistently(testEmitter.done).ShouldNot(Receive())

			accessLogger.Stop()
		})

	})

	Context("with a file", func() {
		It("writes to the log file", func() {
			var fakeFile = new(test_util.FakeFile)

			accessLogger := NewFileAndLoggregatorAccessLogger(fakeFile, nil)
			go accessLogger.Run()
			accessLogger.Log(*CreateAccessLogRecord())

			var payload []byte
			Eventually(func() int {
				n, _ := fakeFile.Read(&payload)
				return n
			}).ShouldNot(Equal(0))
			Ω(string(payload)).To(MatchRegexp("^.*foo.bar.*\n"))

			accessLogger.Stop()
		})
	})

	Context("with valid hostnames", func() {
		It("creates an emitter", func() {
			e, err := NewEmitter("localhost:9843", "secret", 42)
			Ω(err).ToNot(HaveOccurred())
			Ω(e).ToNot(BeNil())

			e, err = NewEmitter("10.10.16.14:9843", "secret", 42)
			Ω(err).ToNot(HaveOccurred())
			Ω(e).ToNot(BeNil())
		})
	})

	Context("when invalid host:port pairs are provided", func() {
		It("does not create an emitter", func() {
			e, err := NewEmitter("this_is_not_a_url", "secret", 42)
			Ω(err).To(HaveOccurred())
			Ω(e).To(BeNil())

			e, err = NewEmitter("localhost", "secret", 42)
			Ω(err).To(HaveOccurred())
			Ω(e).To(BeNil())

			e, err = NewEmitter("10.10.16.14", "secret", 42)
			Ω(err).To(HaveOccurred())
			Ω(e).To(BeNil())

			e, err = NewEmitter("", "secret", 42)
			Ω(err).To(HaveOccurred())
			Ω(e).To(BeNil())
		})
	})

	Measure("Log write speed", func(b Benchmarker) {
		r := CreateAccessLogRecord()
		w := nullWriter{}

		b.Time("writeTime", func() {
			for i := 0; i < 100; i++ {
				r.WriteTo(w)
			}
		})
	}, 100)
})

func CreateAccessLogRecord() *AccessLogRecord {
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

	b := route.NewEndpoint("my_awesome_id", "127.0.0.1", 4567, "", nil)

	r := AccessLogRecord{
		Request:       req,
		StatusCode:    res.StatusCode,
		RouteEndpoint: b,
		StartedAt:     time.Unix(10, 100000000),
		FirstByteAt:   time.Unix(10, 200000000),
		FinishedAt:    time.Unix(10, 300000000),
		BodyBytesSent: 42,
	}

	return &r
}

type nullWriter struct{}

func (n nullWriter) Write(b []byte) (int, error) {
	return len(b), nil
}
