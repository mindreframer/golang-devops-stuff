package ostent
import (
	"fmt"
	"log"
	"net"
	"sync"
	"time"
	"bufio"
	"net/http"
)

type logged struct {
	loggedmap map[string]struct{}
	mutex sync.Mutex
}

type logger struct {
	production bool
	access     *log.Logger
	logged     logged
}

func NewLogged(production bool, access *log.Logger) *logger {
	return &logger{
		production: production,
		access:     access,
		logged: logged{loggedmap: map[string]struct{}{}},
	}
}

func(lg *logger) Constructor(HANDLER http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		lw := &loggedResponseWriter{ResponseWriter: w}
		HANDLER.ServeHTTP(lw, r)

		if lg.production {
			lg.productionLog(start, *lw, r)
			return
		}
		lg.log(start, "", *lw, r)
	})
}

func (lg *logger) productionLog(start time.Time, w loggedResponseWriter, r *http.Request) {
	if !w.statusgood() {
		lg.log(start, "", w, r)
		return
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	log := func() bool {
		lg.logged.mutex.Lock()
		defer lg.logged.mutex.Unlock()

		if _, ok := lg.logged.loggedmap[host]; ok {
			return false
		}
		lg.logged.loggedmap[host] = struct{}{}
		return true
	}()
	if !log {
		return
	}

	tail := fmt.Sprintf("\t;subsequent successful requests from %s will not be logged", host)
	lg.log(start, tail, w, r)
}

var ZEROTIME, _ = time.Parse("15:04:05", "00:00:00")

func(lg *logger) log(start time.Time, tail string, w loggedResponseWriter, r *http.Request) {
	diff := time.Since(start)
	since := ZEROTIME.Add(diff).Format("5.0000s")

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	uri := r.URL.Path // OR r.RequestURI ??
	if r.Form != nil && len(r.Form) > 0 {
		uri += "?"+r.Form.Encode()
	}
	echo := func(s string) string {
		if s == "" {
			return "-"
		}
		return s
	}
	lg.access.Printf("%s - - [%s] %#v %d %d %#v %#v\t;%s%s\n",
		host,
		start.Format("02/Jan/2006:15:04:05 -0700"),
		fmt.Sprintf("%s %s %s", r.Method, uri, r.Proto),
		w.status,
		w.size,
		echo(r.Header.Get("Referer")),
		echo(r.Header.Get("User-Agent")),
		since,
		tail)
}

type loggedResponseWriter struct {
	http.ResponseWriter
	http.Flusher // ?
	status int
	size int
}

func (w loggedResponseWriter) statusgood() bool {
	return (w.status == http.StatusSwitchingProtocols || // 101
			w.status == http.StatusOK                 || // 200
			w.status == http.StatusNotModified)          // 304
}

func (w *loggedResponseWriter) Flush() {
	if f, ok := w.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (w *loggedResponseWriter) WriteHeader(s int) {
	w.ResponseWriter.WriteHeader(s)
	w.status = s
}

func (w *loggedResponseWriter) Write(b []byte) (int, error) {
	if w.status == 0 { // generic approach to Write-ing before WriteHeader call
		w.WriteHeader(http.StatusOK)
	}
	s, err := w.ResponseWriter.Write(b)
	if err == nil {
		w.size += s
	}
	return s, err
}

func (w *loggedResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if hj, ok := w.ResponseWriter.(http.Hijacker); ok {
		return hj.Hijack()
	}
	return nil, nil, fmt.Errorf("the ResponseWriter doesn't support the Hijacker interface")
}
