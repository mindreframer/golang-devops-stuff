package access_log

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cloudfoundry/gorouter/log"
	"github.com/cloudfoundry/gorouter/route"

	"github.com/cloudfoundry/loggregatorlib/emitter"
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
	return float64(r.FinishedAt.UnixNano() - r.StartedAt.UnixNano())/float64(time.Second)
}

func (r *AccessLogRecord) makeRecord() *bytes.Buffer {
	b := &bytes.Buffer{}
	fmt.Fprintf(b, `%s - `, r.Request.Host)
	fmt.Fprintf(b, `[%s] `, r.FormatStartedAt())
	fmt.Fprintf(b, `"%s %s %s" `, r.Request.Method, r.Request.URL.RequestURI(), r.Request.Proto)

	if r.Response == nil {
		fmt.Fprintf(b, "MissingResponseStatusCode ")
	} else {
		fmt.Fprintf(b, `%d `, r.Response.StatusCode)
	}


	fmt.Fprintf(b, `%d `, r.BodyBytesSent)
	fmt.Fprintf(b, `"%s" `, r.FormatRequestHeader("Referer"))
	fmt.Fprintf(b, `"%s" `, r.FormatRequestHeader("User-Agent"))
	fmt.Fprintf(b, `%s `, r.Request.RemoteAddr)

	if r.ResponseTime() < 0 {
		fmt.Fprintf(b, "response_time:MissingFinishedAt ")
	} else {
		fmt.Fprintf(b, `response_time:%.9f `, r.ResponseTime())
	}

	if r.RouteEndpoint == nil {
		fmt.Fprintf(b, "app_id:MissingRouteEndpointApplicationId")
	} else {
		fmt.Fprintf(b, `app_id:%s`, r.RouteEndpoint.ApplicationId)
	}

	fmt.Fprint(b, "\n")
	return b
}

func (r *AccessLogRecord) WriteTo(w io.Writer) (int64, error) {
	recordBuffer := r.makeRecord()
	return recordBuffer.WriteTo(w)
}

func (r *AccessLogRecord) Emit(e emitter.Emitter) {
	if r.RouteEndpoint == nil {
		return
	}

	if r.RouteEndpoint.ApplicationId == "" {
		return
	}
	recordBuffer := r.makeRecord()
	message := recordBuffer.String()
	log.Debugf("Logging to the loggregator: %s", message)
	e.Emit(r.RouteEndpoint.ApplicationId, message)

}
