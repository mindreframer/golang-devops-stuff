package proxy

import (
	"bytes"
	"fmt"
	"github.com/cloudfoundry/gorouter/log"
	"github.com/cloudfoundry/gorouter/route"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/emitter"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

type AccessLogRecord struct {
	Request       *http.Request
	Response      *http.Response
	RouteEndpoint *route.Endpoint
	StartedAt     time.Time
	FirstByteAt   time.Time
	FinishedAt    time.Time
	BodyBytesSent int64
}

var ipAddressRegex, _ = regexp.Compile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(:[0-9]{1,5}){1}$`)
var hostnameRegex, _ = regexp.Compile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])(:[0-9]{1,5}){1}$`)

func (r *AccessLogRecord) FormatStartedAt() string {
	return r.StartedAt.Format("02/01/2006:15:04:05 -0700")
}

func (r *AccessLogRecord) FormatRequestHeader(k string) (v string) {
	v = r.Request.Header.Get(k)
	if v == "" {
		v = "-"
	}
	return
}

func (r *AccessLogRecord) ResponseTime() float64 {
	return float64(r.FinishedAt.UnixNano()-r.StartedAt.UnixNano()) / float64(time.Second)
}

func (r *AccessLogRecord) makeRecord() *bytes.Buffer {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, `%s - `, r.Request.Host)
	fmt.Fprintf(b, `[%s] `, r.FormatStartedAt())
	fmt.Fprintf(b, `"%s %s %s" `, r.Request.Method, r.Request.URL.RequestURI(), r.Request.Proto)
	fmt.Fprintf(b, `%d `, r.Response.StatusCode)
	fmt.Fprintf(b, `%d `, r.BodyBytesSent)
	fmt.Fprintf(b, `"%s" `, r.FormatRequestHeader("Referer"))
	fmt.Fprintf(b, `"%s" `, r.FormatRequestHeader("User-Agent"))
	fmt.Fprintf(b, `%s `, r.Request.RemoteAddr)
	fmt.Fprintf(b, `response_time:%.9f `, r.ResponseTime())
	fmt.Fprintf(b, `app_id:%s`, r.RouteEndpoint.ApplicationId)
	fmt.Fprint(b, "\n")
	return b
}

func (r *AccessLogRecord) WriteTo(w io.Writer) (int64, error) {
	b := r.makeRecord()
	return b.WriteTo(w)
}

func (r *AccessLogRecord) Emit(e emitter.Emitter) {
	if r.RouteEndpoint.ApplicationId != "" {
		b := r.makeRecord()
		message := b.String()
		log.Debugf("Logging to the loggregator: %s", message)
		e.Emit(r.RouteEndpoint.ApplicationId, message)
	}
}

type AccessLogger struct {
	e     emitter.Emitter
	c     chan AccessLogRecord
	w     io.Writer
	index uint
}

func NewAccessLogger(f io.Writer, loggregatorUrl, loggregatorSharedSecret string, index uint) *AccessLogger {
	a := &AccessLogger{
		w:     f,
		c:     make(chan AccessLogRecord, 128),
		index: index,
	}

	if isValidUrl(loggregatorUrl) {
		a.e, _ = emitter.NewEmitter(loggregatorUrl, "RTR", strconv.FormatUint(uint64(index), 10), loggregatorSharedSecret, steno.NewLogger("router.loggregator"))
	} else {
		log.Errorf("Invalid loggregator url %s", loggregatorUrl)
	}

	return a
}

func (x *AccessLogger) Run() {
	for r := range x.c {
		if x.w != nil {
			r.WriteTo(x.w)
		}
		if x.e != nil {
			r.Emit(x.e)
		}
	}
}

func (x *AccessLogger) Stop() {
	close(x.c)
}

func (x *AccessLogger) Log(r AccessLogRecord) {
	x.c <- r
}

func isValidUrl(url string) bool {
	if ipAddressRegex.MatchString(url) || hostnameRegex.MatchString(url) {
		return true
	}
	return false
}
