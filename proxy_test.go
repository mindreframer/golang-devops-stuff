package vulcan

import (
	"encoding/json"
	"fmt"
	"github.com/mailgun/vulcan/backend"
	"github.com/mailgun/vulcan/control/servicecontrol"
	"github.com/mailgun/vulcan/loadbalance"
	"github.com/mailgun/vulcan/loadbalance/roundrobin"
	"github.com/mailgun/vulcan/netutils"
	"github.com/mailgun/vulcan/timeutils"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type ProxySuite struct {
	timeProvider *timeutils.FreezedTime
	backend      *backend.MemoryBackend
	throttler    *Throttler
	authHeaders  http.Header
}

var _ = Suite(&ProxySuite{})

func (s *ProxySuite) SetUpTest(c *C) {
	start := time.Date(2012, 3, 4, 5, 6, 7, 0, time.UTC)
	s.timeProvider = &timeutils.FreezedTime{CurrentTime: start}
	backend, err := backend.NewMemoryBackend(s.timeProvider)
	c.Assert(err, IsNil)
	s.backend = backend
	s.throttler = NewThrottler(s.backend)
	s.authHeaders = http.Header{"Authorization": []string{"Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ=="}}
}

func (s *ProxySuite) Get(c *C, requestUrl string, header http.Header, body string) (*http.Response, []byte) {
	request, _ := http.NewRequest("GET", requestUrl, strings.NewReader(body))
	netutils.CopyHeaders(request.Header, header)
	request.Close = true
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

func (s *ProxySuite) newController(controlUrls []string) *servicecontrol.Client {
	settings := &servicecontrol.Settings{
		Servers:      controlUrls,
		LoadBalancer: roundrobin.NewRoundRobin(s.timeProvider),
	}
	controller, err := servicecontrol.NewClient(settings)
	if err != nil {
		panic(err)
	}
	return controller
}

func (s *ProxySuite) newProxy(controlServers []*httptest.Server, b backend.Backend, l loadbalance.Balancer) *httptest.Server {
	controlUrls := make([]string, len(controlServers))
	for i, controlServer := range controlServers {
		controlUrls[i] = controlServer.URL
	}

	proxySettings := &ProxySettings{
		Controller:       s.newController(controlUrls),
		ThrottlerBackend: b,
		LoadBalancer:     l,
	}

	proxyHandler, err := NewReverseProxy(proxySettings)
	if err != nil {
		panic(err)
	}

	return httptest.NewServer(proxyHandler)
}

// This proxy requires authentication, so Authenticate header is required
func (s *ProxySuite) TestProxyAuthRequired(c *C) {
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(`{"upstreams": [{"url": "%s"}]}`, upstream.URL)))
	})
	defer control.Close()

	proxy := s.newProxy([]*httptest.Server{control}, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()

	response, bodyBytes := s.Get(c, proxy.URL, http.Header{}, "")
	c.Assert(response.StatusCode, Equals, http.StatusProxyAuthRequired)
	c.Assert(string(bodyBytes), Equals, fmt.Sprintf(`{"error":"%s"}`, http.StatusText(http.StatusProxyAuthRequired)))
}

// Proxy denies request
func (s *ProxySuite) TestProxyAccessDenied(c *C) {
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("I am upstream!"))
	})
	defer upstream.Close()

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("Access denied, sorry"))
	})
	defer control.Close()

	proxy := s.newProxy([]*httptest.Server{control}, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()

	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "")
	c.Assert(response.StatusCode, Equals, http.StatusForbidden)
	c.Assert(string(bodyBytes), Equals, "Access denied, sorry")
}

// Success, make sure we've successfully proxied the response
func (s *ProxySuite) TestSuccess(c *C) {
	var queryValues map[string][]string

	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		queryValues = r.URL.Query()
		w.Write([]byte(fmt.Sprintf(`{"upstreams": [{"url": "%s"}]}`, upstream.URL)))
	})
	defer control.Close()

	proxy := s.newProxy([]*httptest.Server{control}, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()
	requestHeaders := http.Header{"X-Custom-Header": []string{"Bla"}}
	netutils.CopyHeaders(requestHeaders, s.authHeaders)
	response, bodyBytes := s.Get(c, proxy.URL, requestHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream")

	//now make sure that control request was correct
	c.Assert(queryValues, NotNil)
	c.Assert(queryValues["username"][0], Equals, "Aladdin")
	c.Assert(queryValues["password"][0], Equals, "open sesame")
	c.Assert(queryValues["protocol"][0], Equals, "HTTP/1.1")
	c.Assert(queryValues["method"][0], Equals, "GET")
	length, err := strconv.Atoi(queryValues["length"][0])
	c.Assert(err, IsNil)
	c.Assert(length, Equals, len("hello!"))

	headers := s.loadJson([]byte(queryValues["headers"][0]))
	value := headers["X-Custom-Header"]
	c.Assert(fmt.Sprintf("%s", value), Equals, "[Bla]")
}

// Success, make sure we've successfully proxied the response in case when
// one of the control servers went down
func (s *ProxySuite) TestSuccessControlFailover(c *C) {
	var queryValues map[string][]string

	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		queryValues = r.URL.Query()
		w.Write([]byte(fmt.Sprintf(`{"upstreams": [{"url": "%s"}]}`, upstream.URL)))
	})
	defer control.Close()

	proxySettings := &ProxySettings{
		Controller:       s.newController([]string{"http://localhost:9999", control.URL}),
		ThrottlerBackend: s.backend,
		LoadBalancer:     roundrobin.NewRoundRobin(s.timeProvider),
	}

	proxyHandler, err := NewReverseProxy(proxySettings)
	if err != nil {
		c.Assert(err, IsNil)
	}
	proxy := httptest.NewServer(proxyHandler)
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

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write(
			[]byte(
				fmt.Sprintf(`{"failover": {"active": true}, "upstreams": [{"url": "http://localhost:9999"}, {"url": "%s"}]}`,
					upstream.URL)))
	})
	defer control.Close()

	proxy := s.newProxy([]*httptest.Server{control}, &backend.FailingBackend{}, roundrobin.NewRoundRobin(s.timeProvider))
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

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(
			fmt.Sprintf(`{"failover": {"active": true, "codes": [410]}, "upstreams": [{"url": "%s"}, {"url": "%s"}]}`,
				upstream.URL,
				upstream2.URL)))
	})
	defer control.Close()

	proxy := s.newProxy([]*httptest.Server{control}, &backend.FailingBackend{}, roundrobin.NewRoundRobin(s.timeProvider))
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

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(`{"failover": {"active": true}, "upstreams": [{"url": "http://localhost:9999"}, {"url": "%s"}]}`, upstream.URL)))
	})
	defer control.Close()

	proxy := s.newProxy([]*httptest.Server{control}, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
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

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(`{"failover": {"active":true}, "upstreams": [{"url": "%s"}, {"url": "%s"}]}`, slowUpstream.URL, upstream.URL)))
	})
	defer control.Close()

	proxySettings := &ProxySettings{
		Controller:       s.newController([]string{control.URL}),
		ThrottlerBackend: s.backend,
		LoadBalancer:     roundrobin.NewRoundRobin(s.timeProvider),
		HttpReadTimeout:  time.Duration(1) * time.Millisecond,
		HttpDialTimeout:  time.Duration(1) * time.Millisecond,
	}
	proxyHandler, err := NewReverseProxy(proxySettings)
	if err != nil {
		c.Assert(err, IsNil)
	}
	proxy := httptest.NewServer(proxyHandler)
	defer proxy.Close()

	response, bodyBytes := s.Post(c, proxy.URL, s.authHeaders, url.Values{"key": {"Value"}, "id": {"123"}})
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm fast upstream!")
	c.Assert(postedValues.Get("key"), Equals, "Value")
	c.Assert(postedValues.Get("id"), Equals, "123")
}

// Make sure upstream headers were added to the request
func (s *ProxySuite) TestUpstreamHeadersAdded(c *C) {
	var customHeaders http.Header

	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		customHeaders = r.Header
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(`{"upstreams": [{"url": "%s", "headers": {"X-Header-A": ["val"], "X-Header-B": ["val2"]}}]}`, upstream.URL)))
	})
	defer control.Close()

	proxy := s.newProxy([]*httptest.Server{control}, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()
	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream")

	// make sure the headers are set
	c.Assert(customHeaders["X-Header-A"][0], Equals, "val")
	c.Assert(customHeaders["X-Header-B"][0], Equals, "val2")
}

// Make sure instructions headers were added to the request
func (s *ProxySuite) TestInstructionsHeadersAdded(c *C) {
	var customHeaders http.Header

	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		customHeaders = r.Header
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(`{"upstreams": [{"url": "%s"}], "headers": {"X-Header-A": ["val"], "X-Header-B": ["val2"]}}`, upstream.URL)))
	})
	defer control.Close()

	proxy := s.newProxy([]*httptest.Server{control}, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()
	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream")

	// make sure the headers are set
	c.Assert(customHeaders["X-Header-A"][0], Equals, "val")
	c.Assert(customHeaders["X-Header-B"][0], Equals, "val2")
}

// Make sure hop headers were removed
func (s *ProxySuite) TestHopHeadersRemoved(c *C) {
	var capturedHeaders http.Header

	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		capturedHeaders = r.Header
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(`{"upstreams": [{"url": "%s"}]}`, upstream.URL)))
	})
	defer control.Close()

	proxy := s.newProxy([]*httptest.Server{control}, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
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

// Make sure we've returned response with valid retry-seconds
func (s *ProxySuite) TestUpstreamThrottled(c *C) {
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	// Upstream is out of capacity, we should be told to be throttled
	s.backend.UpdateCount(upstream.URL, time.Minute, 10)

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(`{"upstreams": [{"url": "%s", "rates": [{"increment": 10, "value": 10, "period": "minute"}]}]}`, upstream.URL)))
	})
	defer control.Close()

	proxy := s.newProxy([]*httptest.Server{control}, s.backend, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()
	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "")
	c.Assert(response.StatusCode, Equals, 429)

	m := s.loadJson(bodyBytes)
	c.Assert(m["retry-seconds"], Equals, float64(53))
}

// Make sure we stil forwarded the request even if the throttler failed
func (s *ProxySuite) TestUpstreamThrottlerDown(c *C) {
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi, I'm upstream!"))
	})
	defer upstream.Close()

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(`{"upstreams": [{"url": "%s", "rates": [{"increment": 10, "value": 10, "period": "minute"}]}]}`, upstream.URL)))
	})
	defer control.Close()

	proxy := s.newProxy([]*httptest.Server{control}, &backend.FailingBackend{}, roundrobin.NewRoundRobin(s.timeProvider))
	defer proxy.Close()
	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream!")
}

func (s *ProxySuite) TestProxyControlServerUnreachableControlServer(c *C) {
	proxySettings := &ProxySettings{
		Controller:       s.newController([]string{"http://localhost:9999"}),
		ThrottlerBackend: s.backend,
		LoadBalancer:     roundrobin.NewRoundRobin(s.timeProvider),
	}

	proxyHandler, err := NewReverseProxy(proxySettings)
	if err != nil {
		c.Assert(err, IsNil)
	}
	proxy := httptest.NewServer(proxyHandler)
	defer proxy.Close()

	response, _ := s.Get(c, proxy.URL, s.authHeaders, "")
	c.Assert(response.StatusCode, Equals, http.StatusInternalServerError)
}

func (s *ProxySuite) TestUpstreamUpstreamIsDown(c *C) {
	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"upstreams": [{"url": "http://localhost:9999", "rates": [{"increment": 10, "value": 10, "period": "minute"}]}]}`))
	})
	defer control.Close()

	proxySettings := &ProxySettings{
		Controller:       s.newController([]string{control.URL}),
		ThrottlerBackend: s.backend,
		LoadBalancer:     roundrobin.NewRoundRobin(s.timeProvider),
	}
	proxyHandler, err := NewReverseProxy(proxySettings)
	if err != nil {
		c.Assert(err, IsNil)
	}
	proxy := httptest.NewServer(proxyHandler)
	defer proxy.Close()

	response, _ := s.Get(c, proxy.URL, s.authHeaders, "")
	c.Assert(response.StatusCode, Equals, http.StatusBadGateway)
}

// Make sure proxy gives up when control server is too slow
func (s *ProxySuite) TestControlServerTimeout(c *C) {
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * time.Duration(100))
		w.Write([]byte(fmt.Sprintf(`{"upstreams": [{"url": "%s"}]}`, upstream.URL)))
	})
	// Do not call Close as it will hang for 100 seconds
	defer control.CloseClientConnections()

	settings := &servicecontrol.Settings{
		Servers:      []string{control.URL},
		LoadBalancer: roundrobin.NewRoundRobin(s.timeProvider),
		ReadTimeout:  time.Duration(1) * time.Millisecond,
		DialTimeout:  time.Duration(1) * time.Millisecond,
	}
	controller, err := servicecontrol.NewClient(settings)
	if err != nil {
		panic(err)
	}
	proxySettings := &ProxySettings{
		Controller:       controller,
		ThrottlerBackend: s.backend,
		LoadBalancer:     roundrobin.NewRoundRobin(s.timeProvider),
	}
	proxyHandler, err := NewReverseProxy(proxySettings)
	if err != nil {
		c.Assert(err, IsNil)
	}
	proxy := httptest.NewServer(proxyHandler)
	defer proxy.Close()

	response, _ := s.Get(c, proxy.URL, s.authHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusInternalServerError)
}

// Make sure proxy fails over when the fist control server times out
func (s *ProxySuite) TestControlServerTimeoutFailover(c *C) {
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Hi, I'm upstream"))
	})
	defer upstream.Close()

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * time.Duration(100))
		w.Write([]byte(fmt.Sprintf(`{"upstreams": [{"url": "%s"}]}`, upstream.URL)))
	})
	// Do not call Close as it will hang for 100 seconds
	defer control.CloseClientConnections()

	control2 := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(`{"upstreams": [{"url": "%s"}]}`, upstream.URL)))
	})
	// Do not call Close as it will hang for 100 seconds
	defer control.CloseClientConnections()

	settings := &servicecontrol.Settings{
		Servers:      []string{control.URL, control2.URL},
		LoadBalancer: roundrobin.NewRoundRobin(s.timeProvider),
		ReadTimeout:  time.Duration(1) * time.Millisecond,
		DialTimeout:  time.Duration(1) * time.Millisecond,
	}
	controller, err := servicecontrol.NewClient(settings)
	if err != nil {
		panic(err)
	}

	proxySettings := &ProxySettings{
		Controller:       controller,
		ThrottlerBackend: s.backend,
		LoadBalancer:     roundrobin.NewRoundRobin(s.timeProvider),
		HttpReadTimeout:  time.Duration(1) * time.Millisecond,
		HttpDialTimeout:  time.Duration(1) * time.Millisecond,
	}
	proxyHandler, err := NewReverseProxy(proxySettings)
	if err != nil {
		c.Assert(err, IsNil)
	}
	proxy := httptest.NewServer(proxyHandler)
	defer proxy.Close()

	response, bodyBytes := s.Get(c, proxy.URL, s.authHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusOK)
	c.Assert(string(bodyBytes), Equals, "Hi, I'm upstream")
}

// The same story with upstream, if upstream is too slow we should give up
func (s *ProxySuite) TestUpstreamServerTimeout(c *C) {
	upstream := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * time.Duration(100))
		w.Write([]byte("Hi, I'm upstream"))
	})
	// Do not call Close as it will hang for 100 seconds
	defer upstream.CloseClientConnections()

	control := s.newServer(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(fmt.Sprintf(`{"upstreams": [{"url": "%s"}]}`, upstream.URL)))
	})
	defer control.Close()

	proxySettings := &ProxySettings{
		Controller:       s.newController([]string{control.URL}),
		ThrottlerBackend: s.backend,
		LoadBalancer:     roundrobin.NewRoundRobin(s.timeProvider),
		HttpReadTimeout:  time.Duration(1) * time.Millisecond,
		HttpDialTimeout:  time.Duration(1) * time.Millisecond,
	}
	proxyHandler, err := NewReverseProxy(proxySettings)
	if err != nil {
		c.Assert(err, IsNil)
	}
	proxy := httptest.NewServer(proxyHandler)
	defer proxy.Close()

	response, _ := s.Get(c, proxy.URL, s.authHeaders, "hello!")
	c.Assert(response.StatusCode, Equals, http.StatusBadGateway)
}
