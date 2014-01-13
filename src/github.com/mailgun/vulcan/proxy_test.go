package vulcan

import (
	"encoding/json"
	"fmt"
	"github.com/mailgun/vulcan/backend"
	"github.com/mailgun/vulcan/control/js"
	"github.com/mailgun/vulcan/loadbalance"
	"github.com/mailgun/vulcan/loadbalance/roundrobin"
	"github.com/mailgun/vulcan/metrics"
	"github.com/mailgun/vulcan/netutils"
	"github.com/mailgun/vulcan/ratelimit"
	"github.com/mailgun/vulcan/timeutils"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"time"
)

type ProxySuite struct {
	timeProvider *timeutils.FreezedTime
	backend      *backend.MemoryBackend
	limiter      ratelimit.RateLimiter
	authHeaders  http.Header
	metrics      metrics.ProxyMetrics
}

var _ = Suite(&ProxySuite{})

func (s *ProxySuite) SetUpTest(c *C) {
	start := time.Date(2012, 3, 4, 5, 6, 7, 0, time.UTC)
	s.timeProvider = &timeutils.FreezedTime{CurrentTime: start}
	backend, err := backend.NewMemoryBackend(s.timeProvider)
	c.Assert(err, IsNil)
	s.backend = backend
	s.limiter = &ratelimit.BasicRateLimiter{Backend: s.backend}
	s.authHeaders = http.Header{"Authorization": []string{"Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ=="}}
}

func (s *ProxySuite) Get(c *C, requestUrl string, header http.Header, body string) (*http.Response, []byte) {
	request, _ := http.NewRequest("GET", requestUrl, strings.NewReader(body))
	netutils.CopyHeaders(request.Header, header)
	request.Close = true
	// the HTTP lib treats Host as a special header.  it only respects the value on req.Host, and ignores
	// values in req.Headers
	if header.Get("Host") != "" {
		request.Host = header.Get("Host")
	}
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		c.Fatalf("Get: %v", err)
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		c.Fatalf("Get body failed: %v", err)
	}
	return response, bodyBytes
}

func (s *ProxySuite) Post(c *C, requestUrl string, header http.Header, body url.Values) (*http.Response, []byte) {
	request, _ := http.NewRequest("POST", requestUrl, strings.NewReader(body.Encode()))
	netutils.CopyHeaders(request.Header, header)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	request.Close = true
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		c.Fatalf("Post: %v", err)
	}

	bodyBytes, err := ioutil.ReadAll(response.Body)
	if err != nil {
		c.Fatalf("Post body failed: %v", err)
	}
	return response, bodyBytes
}

type WebHandler func(http.ResponseWriter, *http.Request)

func (s *ProxySuite) newServer(handler WebHandler) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(handler))
}

func (s *ProxySuite) loadJson(bytes []byte) map[string]interface{} {
	var replyObject interface{}
	err := json.Unmarshal(bytes, &replyObject)
	if err != nil {
		panic(err)
	}
	return replyObject.(map[string]interface{})
}

func (s *ProxySuite) newController(code string) *js.JsController {
	return &js.JsController{
		CodeGetter: js.NewStringGetter(code),
	}
}

func (s *ProxySuite) newProxyWithTimeouts(
	code string,
	b backend.Backend,
	l loadbalance.Balancer,
	readTimeout time.Duration,
	dialTimeout time.Duration) *httptest.Server {

	controller := s.newController(code)

	proxySettings := &ProxySettings{
		Controller:       controller,
		ThrottlerBackend: b,
		LoadBalancer:     l,
		HttpReadTimeout:  readTimeout,
		HttpDialTimeout:  dialTimeout,
	}

	s.metrics = metrics.NewProxyMetrics()

	proxy, err := NewReverseProxy(&s.metrics, proxySettings)
	if err != nil {
		panic(err)
	}
	controller.Client = proxy
	return httptest.NewServer(proxy)

}

func (s *ProxySuite) newProxy(code string, b backend.Backend, l loadbalance.Balancer) *httptest.Server {
	return s.newProxyWithTimeouts(code, b, l, time.Duration(0), time.Duration(0))
}

// Success, make sure we've successfully proxied the response
func (s *ProxySuite) TestSuccess(c *C) {
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	code := fmt.Sprintf(
		`function handle(request){
            if(!request.auth.username || ! request.auth.password)
               return {code: 401, body: {error: "Unauthorized"}};
            return {upstreams: ["%s"]};
         }
     `, upstream.URL)

	proxy := s.newProxy(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()

	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream")
}

func (s *ProxySuite) TestUpstreamGetFailover(c *C) {
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi, I'm upstream!"))
	})
	defer upstream.Close()

	code := fmt.Sprintf(
		`function handle(request){
            return {failover: true, upstreams: ["%s", "%s"]};
         }
     `, "http://localhost:9999", upstream.URL)

	proxy := s.newProxy(code, &backend.FailingBackend{}, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()
	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream!")
}

func (s *ProxySuite) TestUpstreamGetFailoverCodes(c *C) {
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusGone)
		w.Write([]byte("Hi, I'm upstream, but I'm shutting down"))
	})
	defer upstream.Close()

	upstream2 := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi, I'm upstream that you need"))
	})
	defer upstream2.Close()

	code := fmt.Sprintf(
		`function handle(request){
            return {failover: {codes: [410]}, upstreams: ["%s", "%s"]};
         }
     `, upstream.URL, upstream2.URL)

	proxy := s.newProxy(code, &backend.FailingBackend{}, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()
	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream that you need")
}

func (s *ProxySuite) TestFailedUpstreamPostFailover(c *C) {
	var postedValues url.Values
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.ParseForm(), IsNil)
		postedValues = r.PostForm
		w.Write([]byte("Hi, I'm upstream!"))
	})
	defer upstream.Close()

	code := fmt.Sprintf(
		`function handle(request){
            return {failover: true, upstreams: ["%s", "%s"]};
         }
     `, "http://localhost:9999", upstream.URL)

	proxy := s.newProxy(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()

	response, bodyBytes := s.Post(c, proxy.URL, s.authHeaders, url.Values{"key": {"Value"}, "id": {"123"}})
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream!")
	c.Assert(postedValues.Get("key"), Equals, "Value")
	c.Assert(postedValues.Get("id"), Equals, "123")
}

// One of the upstreams consumed the request, but freezed and does not respond
func (s *ProxySuite) TestFailedUpstreamPostTimeoutFailover(c *C) {
	var postedValues url.Values
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		c.Assert(r.ParseForm(), IsNil)
		postedValues = r.PostForm
		w.Write([]byte("Hi, I'm fast upstream!"))
	})
	defer upstream.Close()

	slowUpstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		r.ParseForm()
		time.Sleep(time.Second * time.Duration(100))
		w.Write([]byte("Hi, I'm slow upstream!"))
	})
	defer slowUpstream.CloseClientConnections()

	code := fmt.Sprintf(
		`function handle(request){
            return {failover: true, upstreams: ["%s", "%s"]};
         }
     `, slowUpstream.URL, upstream.URL)

	timeout := time.Duration(10) * time.Millisecond
	proxy := s.newProxyWithTimeouts(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider), timeout, timeout)
	defer proxy.Close()

	response, bodyBytes := s.Post(c, proxy.URL, s.authHeaders, url.Values{"key": {"Value"}, "id": {"123"}})
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm fast upstream!")
	c.Assert(postedValues.Get("key"), Equals, "Value")
	c.Assert(postedValues.Get("id"), Equals, "123")
}

// Make sure upstream headers were added to the request
func (s *ProxySuite) TestHeadersAdded(c *C) {
	var customHeaders http.Header

	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		customHeaders = r.Header
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	code := fmt.Sprintf(
		`function handle(request){
            return {
               upstreams: ["%s"],
               add_headers: {"X-Header-A": ["val"], "X-Header-B": ["val2"]},
            };
         }
     `, upstream.URL)

	proxy := s.newProxy(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()
	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream")

	// make sure the headers are set
	c.Assert(customHeaders["X-Header-A"][0], Equals, "val")
	c.Assert(customHeaders["X-Header-B"][0], Equals, "val2")
}

// Make sure upstream headers were removed from the request
func (s *ProxySuite) TestHeadersRemoved(c *C) {
	var customHeaders http.Header

	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		customHeaders = r.Header
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	code := fmt.Sprintf(
		`function handle(request){
            return {
               upstreams: ["%s"],
               remove_headers: ["x-authorized", "X-Account-id"],
            };
         }
     `, upstream.URL)

	proxy := s.newProxy(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()

	headers := make(http.Header)
	headers.Add("X-Authorized", "yes")
	headers.Add("X-Authorized", "sure")
	headers.Add("X-Account-Id", "a")

	response, bodyBytes := s.Get(c, proxy.URL, headers, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream")

	// make sure the headers are removed
	for key, _ := range headers {
		c.Assert(customHeaders.Get(key), Equals, "")
	}
}

// Make sure hop headers were removed
func (s *ProxySuite) TestHopHeadersRemoved(c *C) {
	var capturedHeaders http.Header

	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	code := fmt.Sprintf(
		`function handle(request){
            return {failover: true, upstreams: ["%s"]};
         }
     `, upstream.URL)

	proxy := s.newProxy(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()

	headers := make(http.Header)
	headers.Add("Connection", "close")
	headers.Add("Keep-Alive", "timeout=600")
	headers.Add("Proxy-Authenticate", "Negotiate")
	headers.Add("Proxy-Authorization", "Basic YW55IGNhcm5hbCBwbGVhcw==")
	headers.Add("Authorization", "Basic YW55IGNhcm5hbCBwbGVhcw==")
	headers.Add("Te", "deflate")
	headers.Add("Trailer", "a")
	headers.Add("Transfer-Encoding", "chunked")
	headers.Add("Upgrade", "IRC/6.9")

	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream")

	// make sure the headers are removed
	for _, h := range hopHeaders {
		c.Assert(capturedHeaders.Get(h), Equals, "")
	}
}

func (s *ProxySuite) TestForwardHeadersAdded(c *C) {
	var capturedHeaders http.Header

	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	code := fmt.Sprintf(
		`function handle(request){
            return {failover: true, upstreams: ["%s"]};
         }
     `, upstream.URL)

	proxy := s.newProxy(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()

	response, bodyBytes := s.Get(c,
		proxy.URL,
		http.Header{"Host": []string{"crazyhostname.example.com"}},
		"hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream")

	hostname, _ := os.Hostname()
	c.Assert(capturedHeaders.Get("X-Forwarded-For"), Equals, "127.0.0.1")
	c.Assert(capturedHeaders.Get("X-Forwarded-Proto"), Equals, "http")
	c.Assert(capturedHeaders.Get("X-Forwarded-Host"), Equals, "crazyhostname.example.com")
	c.Assert(capturedHeaders.Get("X-Forwarded-Server"), Equals, hostname)
}

func (s *ProxySuite) TestProxyAuthRequired(c *C) {
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	code := fmt.Sprintf(
		`function handle(request){
            if(!request.username || ! request.password)
               return {code: 401, body: {error: "Unauthorized"}};
            return {upstreams: ["%s"]};
         }
     `, upstream.URL)

	proxy := s.newProxy(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()

	response, bodyBytes := s.Get(c, proxy.URL, http.Header{}, "")
	c.Assert(response.StatusCode, Equals, http.StatusUnauthorized)
	c.Assert(string(bodyBytes), Equals, fmt.Sprintf(`{"error":"%s"}`, http.StatusText(http.StatusUnauthorized)))
}

// Proxy denies request
func (s *ProxySuite) TestProxyAccessDenied(c *C) {
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("I am upstream!"))
	})
	defer upstream.Close()

	code := `function handle(request){
               return {code: 403, body: {error: "Forbidden"}};
             }`

	proxy := s.newProxy(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()

	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "")
	c.Assert(response.StatusCode, Equals, http.StatusForbidden)
	c.Assert(string(bodyBytes), Equals, `{"error":"Forbidden"}`)
}

// Make sure we've returned response with valid retry-seconds
func (s *ProxySuite) TestRateLimited(c *C) {
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	// Upstream is out of capacity, we should be told to be throttled
	s.backend.UpdateCount("all_requests", time.Minute, 10)

	code := fmt.Sprintf(
		`function handle(request){
            return {upstreams: ["%s"], rates: {all: "10 requests/minute"}};
         }
     `, upstream.URL)

	proxy := s.newProxy(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()

	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "")
	c.Assert(response.StatusCode, Equals, 429)

	m := s.loadJson(bodyBytes)
	c.Assert(m["retry_seconds"], Equals, float64(53))
}

// Make sure we stil forwarded the request even if the rate limiter failed
func (s *ProxySuite) TestUpstreamRateLimiterDown(c *C) {
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi, I'm upstream!"))
	})
	defer upstream.Close()

	code := fmt.Sprintf(
		`function handle(request){
            return {upstreams: ["%s"], rates: {all: "10 requests/minute"}};
         }
     `, upstream.URL)

	proxy := s.newProxy(code, &backend.FailingBackend{}, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()
	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream!")
}

// Make sure we don't panic when all upstreams are down
func (s *ProxySuite) TestUpstreamUpstreamIsDown(c *C) {
	code := `function handle(request){
            return {upstreams: ["http://localhost:9999"]};
         }`
	proxy := s.newProxy(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	response, _ := s.Get(c, proxy.URL, s.authHeaders, "")
	c.Assert(response.StatusCode, Equals, http.StatusBadGateway)
}

// Make sure we failover if one upstream is too slow to respond
func (s *ProxySuite) TestUpstreamServerTimeout(c *C) {
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * time.Duration(100))
		w.Write([]byte("Hi, I'm upstream"))
	})
	// Do not call Close as it will hang for 100 seconds
	defer upstream.CloseClientConnections()

	upstream2 := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi, I'm upstream 2"))
	})
	defer upstream2.Close()

	code := fmt.Sprintf(
		`function handle(request){
            return {failover: true, upstreams: ["%s", "%s"]};
         }
     `, upstream.URL, upstream2.URL)

	timeout := time.Duration(10) * time.Millisecond
	proxy := s.newProxyWithTimeouts(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider), timeout, timeout)
	defer proxy.Close()

	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream 2")
}

// Make sure that path has been altered
func (s *ProxySuite) TestRewritePath(c *C) {
	path := ""

	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		path = r.URL.Path
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	code := fmt.Sprintf(
		`function handle(request){
            return {failover: true, upstreams: ["%s"], rewrite_path: "/new/path"};
         }
     `, upstream.URL)

	proxy := s.newProxy(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()

	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream")
	c.Assert(path, Equals, "/new/path")
}

// Make sure get request in proxy works
func (s *ProxySuite) TestGetRequestInProxyNoParams(c *C) {
	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"response": "hi!"}`))
	})
	defer control.Close()

	code := fmt.Sprintf(
		`function handle(request){
            return get("%s")
         }
     `, control.URL)

	proxy := s.newProxy(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()

	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, `{"response":"hi!"}`)
}

// Make sure get request in proxy works
func (s *ProxySuite) TestGetRequestInProxyQuery(c *C) {
	var query url.Values
	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		query = r.URL.Query()
		w.Write([]byte(`{"response": "hi!"}`))
	})
	defer control.Close()

	code := fmt.Sprintf(
		`function handle(request){
            return get("%s", request.query)
         }
     `, control.URL)

	proxy := s.newProxy(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()

	response, bodyBytes := s.Get(c, fmt.Sprintf("%s?a=b&a=c&x=y", proxy.URL), s.authHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, `{"response":"hi!"}`)
	c.Assert(query, DeepEquals, url.Values{"a": []string{"b", "c"}, "x": []string{"y"}})
}

// Make sure get request in proxy works
func (s *ProxySuite) TestGetRequestInProxyAuth(c *C) {
	var query url.Values
	var headers http.Header
	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		headers = r.Header
		query = r.URL.Query()
		w.Write([]byte(`{"response": "hello!"}`))
	})
	defer control.Close()

	code := fmt.Sprintf(
		`function handle(request){
            return get("%s", request.query, request.auth)
         }
     `, control.URL)

	proxy := s.newProxy(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()

	response, bodyBytes := s.Get(c, fmt.Sprintf("%s?a=b&a=c&x=y", proxy.URL), s.authHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, `{"response":"hello!"}`)
	c.Assert(query, DeepEquals, url.Values{"a": []string{"b", "c"}, "x": []string{"y"}})
	c.Assert(headers.Get("Authorization"), DeepEquals, s.authHeaders.Get("Authorization"))
}

// Make sure get request in proxy works despite of the one server being down
func (s *ProxySuite) TestGetRequestInProxyFailover(c *C) {
	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"response": "hi!"}`))
	})
	defer control.Close()

	code := fmt.Sprintf(
		`function handle(request){
            return get(["http://localhost:9999", "%s"])
         }
     `, control.URL)

	proxy := s.newProxy(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()

	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, `{"response":"hi!"}`)
}

// Make sure get request in proxy works despite of the one server being down
func (s *ProxySuite) TestGetRequestInProxyFailoverOnTimeout(c *C) {
	slowUpstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * time.Duration(100))
		w.Write([]byte(`{"response": "hi, I'm super slow"}`))
	})
	defer slowUpstream.CloseClientConnections()

	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"response": "hi, I'm fast!"}`))
	})
	defer upstream.Close()

	code := fmt.Sprintf(
		`function handle(request){
            return get(["%s", "%s"])
         }
     `, slowUpstream.URL, upstream.URL)

	timeout := time.Duration(10) * time.Millisecond
	proxy := s.newProxyWithTimeouts(code, s.backend, roundrobin.NewRoundRobin(s.timeProvider), timeout, timeout)
	defer proxy.Close()

	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, `{"response":"hi, I'm fast!"}`)
}
