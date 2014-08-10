package proxy_test

import (
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry/gorouter/access_log"
	router_http "github.com/cloudfoundry/gorouter/common/http"
	"github.com/cloudfoundry/gorouter/config"
	"github.com/cloudfoundry/gorouter/registry"
	"github.com/cloudfoundry/gorouter/route"
	"github.com/cloudfoundry/gorouter/stats"
	"github.com/cloudfoundry/gorouter/test_util"
	"github.com/cloudfoundry/yagnats/fakeyagnats"

	. "github.com/cloudfoundry/gorouter/proxy"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const uuid_regex = `^[[:xdigit:]]{8}(-[[:xdigit:]]{4}){3}-[[:xdigit:]]{12}$`

type connHandler func(*test_util.HttpConn)

type nullVarz struct{}

func (_ nullVarz) MarshalJSON() ([]byte, error)                               { return json.Marshal(nil) }
func (_ nullVarz) ActiveApps() *stats.ActiveApps                              { return stats.NewActiveApps() }
func (_ nullVarz) CaptureBadRequest(*http.Request)                            {}
func (_ nullVarz) CaptureBadGateway(*http.Request)                            {}
func (_ nullVarz) CaptureRoutingRequest(b *route.Endpoint, req *http.Request) {}
func (_ nullVarz) CaptureRoutingResponse(b *route.Endpoint, res *http.Response, t time.Time, d time.Duration) {
}

var _ = Describe("Proxy", func() {
	var r *registry.RouteRegistry
	var p Proxy
	var conf *config.Config
	var proxyServer net.Listener
	var accessLog access_log.AccessLogger
	var accessLogFile *test_util.FakeFile

	BeforeEach(func() {
		conf = config.DefaultConfig()
		conf.TraceKey = "my_trace_key"
		conf.EndpointTimeout = 500 * time.Millisecond

		mbus := fakeyagnats.New()

		r = registry.NewRouteRegistry(conf, mbus)

		accessLogFile = new(test_util.FakeFile)
		accessLog = access_log.NewFileAndLoggregatorAccessLogger(accessLogFile, nil)
		go accessLog.Run()

		p = NewProxy(ProxyArgs{
			EndpointTimeout: conf.EndpointTimeout,
			Ip:              conf.Ip,
			TraceKey:        conf.TraceKey,
			Registry:        r,
			Reporter:        nullVarz{},
			AccessLogger:    accessLog,
		})

		ln, err := net.Listen("tcp", "127.0.0.1:0")
		Ω(err).NotTo(HaveOccurred())

		server := http.Server{Handler: p}
		go server.Serve(ln)

		proxyServer = ln
	})

	AfterEach(func() {
		proxyServer.Close()
		accessLog.Stop()
	})

	It("responds to http/1.0", func() {
		ln := registerHandler(r, "test", func(x *test_util.HttpConn) {
			x.CheckLine("GET / HTTP/1.1")

			x.WriteLines([]string{
				"HTTP/1.1 200 OK",
				"Content-Length: 0",
			})
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		x.WriteLines([]string{
			"GET / HTTP/1.0",
			"Host: test",
		})

		x.CheckLine("HTTP/1.0 200 OK")
	})

	It("Logs a request", func() {
		ln := registerHandler(r, "test", func(x *test_util.HttpConn) {
			x.CheckLine("GET / HTTP/1.1")

			x.WriteLines([]string{
				"HTTP/1.1 200 OK",
				"Content-Length: 0",
			})

		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		x.WriteLines([]string{
			"GET / HTTP/1.0",
			"Host: test",
		})

		x.CheckLine("HTTP/1.0 200 OK")

		var payload []byte
		Eventually(func() int {
			accessLogFile.Read(&payload)
			return len(payload)
		}).ShouldNot(BeZero())
		Ω(string(payload)).To(MatchRegexp("^test.*\n"))
		//make sure the record includes all the data
		//since the building of the log record happens throughout the life of the request
		Ω(string(payload)).To(MatchRegexp(".*200.*\n"))
	})

	It("Logs a request when it exits early", func() {
		x := dialProxy(proxyServer)

		x.WriteLines([]string{
			"GET / HTTP/0.9",
			"Host: test",
		})

		x.CheckLine("HTTP/1.0 400 Bad Request")

		var payload []byte
		Eventually(func() int {
			n, e := accessLogFile.Read(&payload)
			Ω(e).ShouldNot(HaveOccurred())
			return n
		}).ShouldNot(BeZero())

		Ω(string(payload)).To(MatchRegexp("^test.*\n"))
	})

	It("responds to HTTP/1.1", func() {
		ln := registerHandler(r, "test", func(x *test_util.HttpConn) {
			x.CheckLine("GET / HTTP/1.1")

			x.WriteLines([]string{
				"HTTP/1.1 200 OK",
				"Content-Length: 0",
			})
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		x.WriteLines([]string{
			"GET / HTTP/1.1",
			"Host: test",
		})

		x.CheckLine("HTTP/1.1 200 OK")
	})

	It("does not respond to unsupported HTTP versions", func() {
		x := dialProxy(proxyServer)

		x.WriteLines([]string{
			"GET / HTTP/0.9",
			"Host: test",
		})

		x.CheckLine("HTTP/1.0 400 Bad Request")
	})

	It("responds to load balancer check", func() {
		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Header.Set("User-Agent", "HTTP-Monitor/1.1")
		x.WriteRequest(req)

		_, body := x.ReadResponse()
		Ω(body).To(Equal("ok\n"))
	})

	It("responds to unknown host with 404", func() {
		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Host = "unknown"
		x.WriteRequest(req)

		resp, body := x.ReadResponse()
		Ω(resp.StatusCode).To(Equal(http.StatusNotFound))
		Ω(resp.Header.Get("X-Cf-RouterError")).To(Equal("unknown_route"))
		Ω(body).To(Equal("404 Not Found: Requested route ('unknown') does not exist.\n"))
	})

	It("responds to misbehaving host with 502", func() {
		ln := registerHandler(r, "enfant-terrible", func(x *test_util.HttpConn) {
			x.Close()
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Host = "enfant-terrible"
		x.WriteRequest(req)

		resp, body := x.ReadResponse()
		Ω(resp.StatusCode).To(Equal(http.StatusBadGateway))
		Ω(resp.Header.Get("X-Cf-RouterError")).To(Equal("endpoint_failure"))
		Ω(body).To(Equal("502 Bad Gateway: Registered endpoint failed to handle the request.\n"))
	})

	It("trace headers added on correct TraceKey", func() {
		ln := registerHandler(r, "trace-test", func(x *test_util.HttpConn) {
			_, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			resp := test_util.NewResponse(http.StatusOK)
			x.WriteResponse(resp)
			x.Close()
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Host = "trace-test"
		req.Header.Set(router_http.VcapTraceHeader, "my_trace_key")
		x.WriteRequest(req)

		resp, _ := x.ReadResponse()
		Ω(resp.StatusCode).To(Equal(http.StatusOK))
		Ω(resp.Header.Get(router_http.VcapBackendHeader)).To(Equal(ln.Addr().String()))
		Ω(resp.Header.Get(router_http.CfRouteEndpointHeader)).To(Equal(ln.Addr().String()))
		Ω(resp.Header.Get(router_http.VcapRouterHeader)).To(Equal(conf.Ip))
	})

	It("trace headers not added on incorrect TraceKey", func() {
		ln := registerHandler(r, "trace-test", func(x *test_util.HttpConn) {
			_, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			resp := test_util.NewResponse(http.StatusOK)
			x.WriteResponse(resp)
			x.Close()
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Host = "trace-test"
		req.Header.Set(router_http.VcapTraceHeader, "a_bad_trace_key")
		x.WriteRequest(req)

		resp, _ := x.ReadResponse()
		Ω(resp.Header.Get(router_http.VcapBackendHeader)).To(Equal(""))
		Ω(resp.Header.Get(router_http.CfRouteEndpointHeader)).To(Equal(""))
		Ω(resp.Header.Get(router_http.VcapRouterHeader)).To(Equal(""))
	})

	It("X-Forwarded-For is added", func() {
		done := make(chan bool)

		ln := registerHandler(r, "app", func(x *test_util.HttpConn) {
			req, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			resp := test_util.NewResponse(http.StatusOK)
			x.WriteResponse(resp)
			x.Close()

			done <- req.Header.Get("X-Forwarded-For") == "127.0.0.1"
		})
		defer ln.Close()

		x := dialProxy(proxyServer)
		req := x.NewRequest("GET", "/", nil)
		req.Host = "app"
		x.WriteRequest(req)

		var answer bool
		Eventually(done).Should(Receive(&answer))
		Ω(answer).To(BeTrue())

		x.ReadResponse()
	})

	It("X-Forwarded-For is appended", func() {
		done := make(chan bool)

		ln := registerHandler(r, "app", func(x *test_util.HttpConn) {
			req, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			resp := test_util.NewResponse(http.StatusOK)
			x.WriteResponse(resp)
			x.Close()

			done <- req.Header.Get("X-Forwarded-For") == "1.2.3.4, 127.0.0.1"
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Host = "app"
		req.Header.Add("X-Forwarded-For", "1.2.3.4")
		x.WriteRequest(req)

		var answer bool
		Eventually(done).Should(Receive(&answer))
		Ω(answer).To(BeTrue())

		x.ReadResponse()
	})

	It("X-Request-Start is appended", func() {
		done := make(chan string)

		ln := registerHandler(r, "app", func(x *test_util.HttpConn) {
			req, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			resp := test_util.NewResponse(http.StatusOK)
			x.WriteResponse(resp)
			x.Close()

			done <- req.Header.Get("X-Request-Start")
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Host = "app"
		x.WriteRequest(req)

		var answer string
		Eventually(done).Should(Receive(&answer))
		Ω(answer).To(MatchRegexp("^\\d{10}\\d{3}$")) // unix timestamp millis

		x.ReadResponse()
	})

	It("X-Request-Start is not overwritten", func() {
		done := make(chan []string)

		ln := registerHandler(r, "app", func(x *test_util.HttpConn) {
			req, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			resp := test_util.NewResponse(http.StatusOK)
			x.WriteResponse(resp)
			x.Close()

			done <- req.Header[http.CanonicalHeaderKey("X-Request-Start")]
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Host = "app"
		req.Header.Add("X-Request-Start", "") // impl cannot just check for empty string
		req.Header.Add("X-Request-Start", "user-set2")
		x.WriteRequest(req)

		var answer []string
		Eventually(done).Should(Receive(&answer))
		Ω(answer).To(Equal([]string{"", "user-set2"}))

		x.ReadResponse()
	})

	It("X-VcapRequest-Id header is added", func() {
		done := make(chan string)

		ln := registerHandler(r, "app", func(x *test_util.HttpConn) {
			req, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			resp := test_util.NewResponse(http.StatusOK)
			x.WriteResponse(resp)
			x.Close()

			done <- req.Header.Get(router_http.VcapRequestIdHeader)
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Host = "app"
		x.WriteRequest(req)

		var answer string
		Eventually(done).Should(Receive(&answer))
		Ω(answer).To(MatchRegexp(uuid_regex))

		x.ReadResponse()
	})

	It("X-Vcap-Request-Id header is overwritten", func() {
		done := make(chan string)

		ln := registerHandler(r, "app", func(x *test_util.HttpConn) {
			req, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			resp := test_util.NewResponse(http.StatusOK)
			x.WriteResponse(resp)
			x.Close()

			done <- req.Header.Get(router_http.VcapRequestIdHeader)
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Host = "app"
		req.Header.Add(router_http.VcapRequestIdHeader, "A-BOGUS-REQUEST-ID")
		x.WriteRequest(req)

		var answer string
		Eventually(done).Should(Receive(&answer))
		Ω(answer).ToNot(Equal("A-BOGUS-REQUEST-ID"))
		Ω(answer).To(MatchRegexp(uuid_regex))

		x.ReadResponse()
	})

	It("X-CF-InstanceID header is added literally if present in the routing endpoint", func() {
		done := make(chan string)

		ln := registerHandlerWithInstanceId(r, "app", func(x *test_util.HttpConn) {
			req, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			resp := test_util.NewResponse(http.StatusOK)
			x.WriteResponse(resp)
			x.Close()

			done <- req.Header.Get(router_http.CfInstanceIdHeader)
		}, "fake-instance-id")
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Host = "app"
		x.WriteRequest(req)

		var answer string
		Eventually(done).Should(Receive(&answer))
		Ω(answer).To(Equal("fake-instance-id"))

		x.ReadResponse()
	})

	It("X-CF-InstanceID header is added with host:port information if NOT present in the routing endpoint", func() {
		done := make(chan string)

		ln := registerHandler(r, "app", func(x *test_util.HttpConn) {
			req, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			resp := test_util.NewResponse(http.StatusOK)
			x.WriteResponse(resp)
			x.Close()

			done <- req.Header.Get(router_http.CfInstanceIdHeader)
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Host = "app"
		x.WriteRequest(req)

		var answer string
		Eventually(done).Should(Receive(&answer))
		Ω(answer).To(MatchRegexp(`^\d+(\.\d+){3}:\d+$`))

		x.ReadResponse()
	})

	It("upgrades for a WebSocket request", func() {
		done := make(chan bool)

		ln := registerHandler(r, "ws", func(x *test_util.HttpConn) {
			req, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			done <- req.Header.Get("Upgrade") == "WebsockeT" &&
				req.Header.Get("Connection") == "UpgradE"

			resp := test_util.NewResponse(http.StatusSwitchingProtocols)
			resp.Header.Set("Upgrade", "WebsockeT")
			resp.Header.Set("Connection", "UpgradE")

			x.WriteResponse(resp)

			x.CheckLine("hello from client")
			x.WriteLine("hello from server")
			x.Close()
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/chat", nil)
		req.Host = "ws"
		req.Header.Set("Upgrade", "WebsockeT")
		req.Header.Set("Connection", "UpgradE")

		x.WriteRequest(req)

		var answer bool
		Eventually(done).Should(Receive(&answer))
		Ω(answer).To(BeTrue())

		resp, _ := x.ReadResponse()
		Ω(resp.StatusCode).To(Equal(http.StatusSwitchingProtocols))
		Ω(resp.Header.Get("Upgrade")).To(Equal("WebsockeT"))
		Ω(resp.Header.Get("Connection")).To(Equal("UpgradE"))

		x.WriteLine("hello from client")
		x.CheckLine("hello from server")

		x.Close()
	})

	It("upgrades for a WebSocket request with comma-separated Connection header", func() {
		done := make(chan bool)

		ln := registerHandler(r, "ws-cs-header", func(x *test_util.HttpConn) {
			req, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			done <- req.Header.Get("Upgrade") == "Websocket" &&
				req.Header.Get("Connection") == "keep-alive, Upgrade"

			resp := test_util.NewResponse(http.StatusSwitchingProtocols)
			resp.Header.Set("Upgrade", "Websocket")
			resp.Header.Set("Connection", "Upgrade")

			x.WriteResponse(resp)

			x.CheckLine("hello from client")
			x.WriteLine("hello from server")
			x.Close()
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/chat", nil)
		req.Host = "ws-cs-header"
		req.Header.Add("Upgrade", "Websocket")
		req.Header.Add("Connection", "keep-alive, Upgrade")

		x.WriteRequest(req)

		var answer bool
		Eventually(done).Should(Receive(&answer))
		Ω(answer).To(BeTrue())

		resp, _ := x.ReadResponse()
		Ω(resp.StatusCode).To(Equal(http.StatusSwitchingProtocols))

		Ω(resp.Header.Get("Upgrade")).To(Equal("Websocket"))
		Ω(resp.Header.Get("Connection")).To(Equal("Upgrade"))

		x.WriteLine("hello from client")
		x.CheckLine("hello from server")

		x.Close()
	})

	It("upgrades for a WebSocket request with multiple Connection headers", func() {
		done := make(chan bool)

		ln := registerHandler(r, "ws-cs-header", func(x *test_util.HttpConn) {
			req, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			done <- req.Header.Get("Upgrade") == "Websocket" &&
				req.Header[http.CanonicalHeaderKey("Connection")][0] == "keep-alive" &&
				req.Header[http.CanonicalHeaderKey("Connection")][1] == "Upgrade"

			resp := test_util.NewResponse(http.StatusSwitchingProtocols)
			resp.Header.Set("Upgrade", "Websocket")
			resp.Header.Set("Connection", "Upgrade")

			x.WriteResponse(resp)

			x.CheckLine("hello from client")
			x.WriteLine("hello from server")
			x.Close()
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/chat", nil)
		req.Host = "ws-cs-header"
		req.Header.Add("Upgrade", "Websocket")
		req.Header.Add("Connection", "keep-alive")
		req.Header.Add("Connection", "Upgrade")

		x.WriteRequest(req)

		var answer bool
		Eventually(done).Should(Receive(&answer))
		Ω(answer).To(BeTrue())

		resp, _ := x.ReadResponse()
		Ω(resp.StatusCode).To(Equal(http.StatusSwitchingProtocols))

		Ω(resp.Header.Get("Upgrade")).To(Equal("Websocket"))
		Ω(resp.Header.Get("Connection")).To(Equal("Upgrade"))

		x.WriteLine("hello from client")
		x.CheckLine("hello from server")

		x.Close()
	})

	It("upgrades a Tcp request", func() {
		ln := registerHandler(r, "tcp-handler", func(x *test_util.HttpConn) {
			x.WriteLine("hello")
			x.CheckLine("hello from client")
			x.WriteLine("hello from server")
			x.Close()
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/chat", nil)
		req.Host = "tcp-handler"
		req.Header.Set("Upgrade", "tcp")

		req.Header.Set("Connection", "UpgradE")

		x.WriteRequest(req)

		x.CheckLine("hello")
		x.WriteLine("hello from client")
		x.CheckLine("hello from server")

		x.Close()
	})

	It("transfers chunked encodings", func() {
		ln := registerHandler(r, "chunk", func(x *test_util.HttpConn) {
			r, w := io.Pipe()

			// Write 3 times on a 100ms interval
			go func() {
				t := time.NewTicker(100 * time.Millisecond)
				defer t.Stop()
				defer w.Close()

				for i := 0; i < 3; i++ {
					<-t.C
					_, err := w.Write([]byte("hello"))
					Ω(err).NotTo(HaveOccurred())
				}
			}()

			_, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			resp := test_util.NewResponse(http.StatusOK)
			resp.TransferEncoding = []string{"chunked"}
			resp.Body = r
			resp.Write(x)
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Host = "chunk"

		err := req.Write(x)
		Ω(err).NotTo(HaveOccurred())

		resp, err := http.ReadResponse(x.Reader, &http.Request{})
		Ω(err).NotTo(HaveOccurred())

		Ω(resp.StatusCode).To(Equal(http.StatusOK))
		Ω(resp.TransferEncoding).To(Equal([]string{"chunked"}))

		// Expect 3 individual reads to complete
		b := make([]byte, 16)
		for i := 0; i < 3; i++ {
			n, err := resp.Body.Read(b[0:])
			if err != nil {
				Ω(err).To(Equal(io.EOF))
			}
			Ω(n).To(Equal(5))
			Ω(string(b[0:n])).To(Equal("hello"))
		}
	})

	It("status no content was no Transfer Encoding response header", func() {
		ln := registerHandler(r, "not-modified", func(x *test_util.HttpConn) {
			_, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			resp := test_util.NewResponse(http.StatusNoContent)
			resp.Header.Set("Connection", "close")
			x.WriteResponse(resp)
			x.Close()
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)

		req.Header.Set("Connection", "close")
		req.Host = "not-modified"
		x.WriteRequest(req)

		resp, _ := x.ReadResponse()
		Ω(resp.StatusCode).To(Equal(http.StatusNoContent))
		Ω(resp.TransferEncoding).To(BeNil())
	})

	It("handles encoded requests", func() {
		ln := registerHandler(r, "encoding", func(x *test_util.HttpConn) {
			x.CheckLine("GET /hello+world?inline-depth=1 HTTP/1.1")
			resp := test_util.NewResponse(http.StatusOK)
			x.WriteResponse(resp)
			x.Close()
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/hello%2bworld?inline-depth=1", nil)
		req.Host = "encoding"
		x.WriteRequest(req)
		resp, _ := x.ReadResponse()

		Ω(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("conforms to the http spec by not encoding ! characters in path", func() {
		ln := registerHandler(r, "encoding", func(x *test_util.HttpConn) {
			x.CheckLine("GET /hello!world?inline-depth=1 HTTP/1.1")
			resp := test_util.NewResponse(http.StatusOK)
			x.WriteResponse(resp)
			x.Close()
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/hello!world?inline-depth=1", nil)
		req.Host = "encoding"
		x.WriteRequest(req)
		resp, _ := x.ReadResponse()

		Ω(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("handles requests with encoded query strings", func() {
		ln := registerHandler(r, "query", func(x *test_util.HttpConn) {
			x.CheckLine("GET /test?a=b&b%3D+bc+&c%3Dd%26e HTTP/1.1")

			resp := test_util.NewResponse(http.StatusOK)
			x.WriteResponse(resp)
			x.Close()
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		queryString := strings.Join([]string{"a=b", url.QueryEscape("b= bc "), url.QueryEscape("c=d&e")}, "&")
		req := x.NewRequest("GET", "/test?"+queryString, nil)
		req.Host = "query"
		x.WriteRequest(req)
		resp, _ := x.ReadResponse()

		Ω(resp.StatusCode).To(Equal(http.StatusOK))
	})

	It("request terminates with slow response", func() {
		ln := registerHandler(r, "slow-app", func(x *test_util.HttpConn) {
			_, err := http.ReadRequest(x.Reader)
			Ω(err).NotTo(HaveOccurred())

			time.Sleep(1 * time.Second)
			resp := test_util.NewResponse(http.StatusOK)
			x.WriteResponse(resp)
			x.Close()
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Host = "slow-app"

		started := time.Now()
		x.WriteRequest(req)

		resp, _ := x.ReadResponse()
		Ω(resp.StatusCode).To(Equal(http.StatusBadGateway))
		Ω(time.Since(started)).To(BeNumerically("<", time.Duration(800*time.Millisecond)))
	})

	It("proxy detects closed client connection", func() {
		serverResult := make(chan error)
		ln := registerHandler(r, "slow-app", func(x *test_util.HttpConn) {
			x.CheckLine("GET / HTTP/1.1")

			timesToTick := 10

			x.WriteLines([]string{
				"HTTP/1.1 200 OK",
				fmt.Sprintf("Content-Length: %d", timesToTick),
			})

			for i := 0; i < 10; i++ {
				_, err := x.Conn.Write([]byte("x"))
				if err != nil {
					serverResult <- err
					return
				}

				time.Sleep(100 * time.Millisecond)
			}

			serverResult <- nil
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Host = "slow-app"
		x.WriteRequest(req)

		x.Conn.Close()

		var err error
		Eventually(serverResult).Should(Receive(&err))
		Ω(err).NotTo(BeNil())
	})

	It("disables keepalives from clients -- force connection close", func() {
		ln := registerHandler(r, "remote", func(x *test_util.HttpConn) {
			http.ReadRequest(x.Reader)
			resp := test_util.NewResponse(http.StatusOK)
			resp.Header.Set("Connection", "keep-alive")
			x.WriteResponse(resp)
			x.Close()
		})
		defer ln.Close()

		x := dialProxy(proxyServer)

		req := x.NewRequest("GET", "/", nil)
		req.Host = "remote"
		x.WriteRequest(req)
		resp, _ := x.ReadResponse()
		Ω(resp.StatusCode).To(Equal(http.StatusOK))

		x.WriteRequest(req)
		_, err := http.ReadResponse(x.Reader, &http.Request{})
		Ω(err).Should(HaveOccurred())
	})

	It("retries when failed endpoints exist", func() {
		ln := registerHandler(r, "retries", func(x *test_util.HttpConn) {
			x.CheckLine("GET / HTTP/1.1")
			resp := test_util.NewResponse(http.StatusOK)
			x.WriteResponse(resp)
			x.Close()
		})
		defer ln.Close()

		ip, err := net.ResolveTCPAddr("tcp", "localhost:81")
		Ω(err).Should(BeNil())
		registerAddr(r, "retries", ip, "instanceId")

		for i := 0; i < 5; i++ {
			x := dialProxy(proxyServer)

			req := x.NewRequest("GET", "/", nil)
			req.Host = "retries"
			x.WriteRequest(req)
			resp, _ := x.ReadResponse()

			Ω(resp.StatusCode).To(Equal(http.StatusOK))
		}
	})

	Context("Wait", func() {
		It("waits for requests to finish", func() {
			blocker := make(chan bool)
			ln := registerHandler(r, "waitforme", func(x *test_util.HttpConn) {
				x.CheckLine("GET /whatever HTTP/1.1")

				blocker <- true
				<-blocker

				resp := test_util.NewResponse(http.StatusOK)
				x.WriteResponse(resp)
				x.Close()
			})
			defer ln.Close()

			x := dialProxy(proxyServer)
			req := x.NewRequest("GET", "/whatever", nil)
			req.Host = "waitforme"
			x.WriteRequest(req)

			<-blocker

			doneWaiting := make(chan struct{})
			go func() {
				p.Wait()
				close(doneWaiting)
			}()

			Consistently(doneWaiting).ShouldNot(BeClosed())

			close(blocker)

			resp, _ := x.ReadResponse()

			Ω(resp.StatusCode).To(Equal(http.StatusOK))
			Eventually(doneWaiting).Should(BeClosed())
		})
	})
})

func registerAddr(r *registry.RouteRegistry, u string, a net.Addr, instanceId string) {
	h, p, err := net.SplitHostPort(a.String())
	Ω(err).NotTo(HaveOccurred())

	x, err := strconv.Atoi(p)
	Ω(err).NotTo(HaveOccurred())

	r.Register(route.Uri(u), route.NewEndpoint("", h, uint16(x), instanceId, nil))
}

func registerHandler(r *registry.RouteRegistry, u string, h connHandler) net.Listener {
	return registerHandlerWithInstanceId(r, u, h, "")
}

func registerHandlerWithInstanceId(r *registry.RouteRegistry, u string, h connHandler, instanceId string) net.Listener {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	Ω(err).NotTo(HaveOccurred())

	go func() {
		var tempDelay time.Duration // how long to sleep on accept failure
		for {
			conn, err := ln.Accept()
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Temporary() {
					if tempDelay == 0 {
						tempDelay = 5 * time.Millisecond
					} else {
						tempDelay *= 2
					}
					if max := 1 * time.Second; tempDelay > max {
						tempDelay = max
					}
					fmt.Printf("http: Accept error: %v; retrying in %v\n", err, tempDelay)
					time.Sleep(tempDelay)
					continue
				}
				break
			}
			go func() {
				defer GinkgoRecover()
				h(test_util.NewHttpConn(conn))
			}()
		}
	}()

	registerAddr(r, u, ln.Addr(), instanceId)

	return ln
}

func dialProxy(proxyServer net.Listener) *test_util.HttpConn {
	x, err := net.Dial("tcp", proxyServer.Addr().String())
	Ω(err).NotTo(HaveOccurred())

	return test_util.NewHttpConn(x)
}
