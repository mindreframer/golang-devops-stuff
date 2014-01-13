// This package contains the proxy core - the main proxy function that accepts and modifies
// request, forwards or denies it.
package vulcan

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/mailgun/vulcan/backend"
	"github.com/mailgun/vulcan/client"
	"github.com/mailgun/vulcan/command"
	"github.com/mailgun/vulcan/control"
	"github.com/mailgun/vulcan/loadbalance"
	"github.com/mailgun/vulcan/metrics"
	"github.com/mailgun/vulcan/netutils"
	"github.com/mailgun/vulcan/ratelimit"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// Reverse proxy settings, what loadbalancing algo to use,
// timeouts, rate limiting backend
type ProxySettings struct {
	// Controlller tells proxy what to do with each request
	Controller control.Controller
	// MemoryBackend or CassandraBackend
	ThrottlerBackend backend.Backend
	// Load balancing algo, e.g. RandomLoadBalancer
	LoadBalancer loadbalance.Balancer
	// How long would proxy wait for server response
	HttpReadTimeout time.Duration
	// How long would proxy try to dial server
	HttpDialTimeout time.Duration
}

// This is a reverse proxy, not meant to be created directly,
// use NewReverseProxy function instead
type ReverseProxy struct {
	// Metrics we track about this reverse proxy.
	metrics *metrics.ProxyMetrics
	// Controller decides what to do with the request
	controller control.Controller
	// Load balancer algorightm implementation
	loadBalancer loadbalance.Balancer
	// Customized transport with dial and read timeouts set
	httpTransport *http.Transport
	// Client that uses customized transport
	httpClient *http.Client
	// Rate limiter
	rateLimiter ratelimit.RateLimiter
}

// Standard dial and read timeouts, can be overriden when supplying proxy settings
const (
	DefaultHttpReadTimeout = time.Duration(10) * time.Second
	DefaultHttpDialTimeout = time.Duration(10) * time.Second
)

// Hop-by-hop headers. These are removed when sent to the backend.
// http://www.w3.org/Protocols/rfc2616/rfc2616-sec13.html
// Copied from reverseproxy.go, too bad
var hopHeaders = []string{
	"Connection",
	"Keep-Alive",
	"Proxy-Authenticate",
	"Proxy-Authorization",
	"Te", // canonicalized version of "TE"
	"Trailers",
	"Transfer-Encoding",
	"Upgrade",
}

// Creates reverse proxy that acts like http server.
func NewReverseProxy(metrics *metrics.ProxyMetrics, s *ProxySettings) (*ReverseProxy, error) {
	s, err := validateProxySettings(s)
	if err != nil {
		return nil, err
	}

	transport := &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, s.HttpDialTimeout)
		},
		ResponseHeaderTimeout: s.HttpReadTimeout,
	}

	var rateLimiter ratelimit.RateLimiter
	if s.ThrottlerBackend != nil {
		rateLimiter = &ratelimit.BasicRateLimiter{Backend: s.ThrottlerBackend}
	}

	p := &ReverseProxy{
		metrics:       metrics,
		controller:    s.Controller,
		loadBalancer:  s.LoadBalancer,
		httpTransport: transport,
		rateLimiter:   rateLimiter,
		httpClient: &http.Client{
			Transport: transport,
		},
	}
	return p, nil
}

// Vulcan implements Getter interface that is used by controllers to issue concurrent get requests with failover
func (p *ReverseProxy) Get(w http.ResponseWriter, hosts []string, query client.MultiDict, auth *netutils.BasicAuth) error {
	req, err := http.NewRequest("GET", "http://localhost", nil)
	if err != nil {
		return err
	}
	if query != nil {
		parameters := url.Values{}
		for key, values := range query {
			for i, _ := range values {
				parameters.Add(key, values[i])
			}
		}
		req.URL.RawQuery = parameters.Encode()
	}

	if auth != nil {
		req.SetBasicAuth(auth.Username, auth.Password)
	}

	upstreams, err := command.NewUpstreamsFromUrls(hosts)
	if err != nil {
		return err
	}
	req.Body = &Buffer{&bytes.Reader{}}

	cmd := &command.Forward{
		Failover:  &command.Failover{Active: true},
		Upstreams: upstreams,
	}
	endpoints := command.EndpointsFromUpstreams(cmd.Upstreams)
	_, err = p.proxyRequest(w, req, cmd, endpoints)
	return err
}

// Main request handler, accepts requests, round trips it to the upstream
// proxies back the response.
func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	p.metrics.Requests.Mark(1)
	glog.Infof("Serving Request %s %s", req.Method, req.RequestURI)

	// Ask controller for instructions
	cmdI, err := p.controller.GetInstructions(req)
	if err != nil {
		glog.Errorf("Error getting instructions: %s", err)
		p.replyError(err, w, req)
		return
	}

	switch cmd := cmdI.(type) {
	case *command.Reply:
		// reply command provides the exact response
		// for the client. Proxy responds and hangs up.
		glog.Infof("Got Reply command: %v", cmd)
		p.metrics.CmdReply.Mark(1)
		p.replyCommand(cmd, w, req)
		return
	case *command.Forward:
		// Forward command contains list of upstreams
		// instructions for the failover and request modification
		glog.Infof("Got Forward command %v", cmd)
		p.metrics.CmdForward.Mark(1)
		// Get upstreams ready to process the request
		retrySeconds, err := p.rateLimit(cmd)
		if err != nil {
			p.replyError(err, w, req)
			return
		}
		if retrySeconds != 0 {
			p.replyError(&command.RetryError{Seconds: retrySeconds}, w, req)
			return
		}
		endpoints := command.EndpointsFromUpstreams(cmd.Upstreams)
		requestBytes, err := p.proxyRequest(w, req, cmd, endpoints)
		if err != nil {
			glog.Error("Failed to proxy to all upstreams:", err)
			p.replyError(&command.AllUpstreamsDownError{}, w, req)
			return
		}
		p.updateRates(requestBytes, cmd)
		return
	}
	p.replyError(fmt.Errorf("Internal logic error"), w, req)
}

func (p *ReverseProxy) rateLimit(cmd *command.Forward) (int, error) {
	if p.rateLimiter == nil || cmd.Rates == nil {
		return 0, nil
	}
	retrySeconds, err := p.rateLimiter.GetRetrySeconds(cmd.Rates)
	// Vulcan prefers to proxy the request in case of rate limiter failure
	// versus hanging up with the error as it's less evil.
	if err != nil {
		glog.Errorf("RateLimiter get stats failure: %s, ignoring error", err)
		return 0, nil
	}
	return retrySeconds, err
}

func (p *ReverseProxy) updateRates(requestBytes int64, cmd *command.Forward) {
	if p.rateLimiter == nil || cmd.Rates == nil {
		return
	}
	err := p.rateLimiter.UpdateStats(requestBytes, cmd.Rates)
	if err != nil {
		glog.Errorf("RateLimiter update stats failire: %s, ignoring error", err)
	}
}

// We need this struct to add a Close method and comply with io.ReadCloser
type Buffer struct {
	*bytes.Reader
}

func (*Buffer) Close() error {
	// Does nothing, created to comply with io.ReadCloser requirements
	return nil
}

func (p *ReverseProxy) nextEndpoint(endpoints []loadbalance.Endpoint) (*command.Endpoint, error) {
	// Get first endpoint
	pendpoint, err := p.loadBalancer.NextEndpoint(endpoints)
	if err != nil {
		glog.Errorf("Loadbalancer failure: %s", err)
		return nil, err
	}
	endpoint, ok := pendpoint.(*command.Endpoint)
	if !ok {
		return nil, fmt.Errorf("Failed to convert types! Unknown type: %v", pendpoint)
	}
	return endpoint, nil
}

// Round trips the request to one of the upstreams, returns the streamed
// request body length in bytes and the upstream reply.
func (p *ReverseProxy) proxyRequest(
	w http.ResponseWriter, req *http.Request,
	cmd *command.Forward,
	endpoints []loadbalance.Endpoint) (int64, error) {

	// We are allowed to fallback in case of upstream failure,
	// record the request body so we can replay it on errors.
	body, err := netutils.NewBodyBuffer(req.Body)
	if err != nil {
		glog.Errorf("Request read error %s", err)
		return 0, netutils.NewHttpError(http.StatusBadRequest)
	}

	requestLength, err := body.TotalSize()
	if err != nil {
		glog.Errorf("Failed to read stored body length: %s", err)
		return 0, netutils.NewHttpError(http.StatusInternalServerError)
	}

	p.metrics.RequestBodySize.Update(requestLength)
	req.Body = body
	defer body.Close()

	for i := 0; i < len(endpoints); i++ {
		_, err := body.Seek(0, 0)
		if err != nil {
			return 0, err
		}
		endpoint, err := p.nextEndpoint(endpoints)
		if err != nil {
			glog.Errorf("Load Balancer failure: %s", err)
			return 0, err
		}
		glog.Infof("With failover, proxy to upstream: %s", endpoint.Upstream)
		err = p.proxyToUpstream(w, req, cmd, endpoint.Upstream)
		if err != nil {
			if cmd.Failover == nil || !cmd.Failover.Active {
				return 0, err
			}
			glog.Errorf("Upstream: %s error: %s, falling back to another", endpoint.Upstream, err)
			// Mark the endpoint as inactive for the next round of the load balance iteration
			endpoint.Active = false
		} else {
			return 0, nil
		}
	}
	glog.Errorf("All upstreams failed!")
	return requestLength, netutils.NewHttpError(http.StatusBadGateway)
}

// Proxy the request to the given upstream, in case if upstream is down
// or failover code sequence has been recorded as the reply, return the error.
// Failover sequence - is a special response code from the upstream that indicates
// that upstream is shutting down and is not willing to accept new requests.
func (p *ReverseProxy) proxyToUpstream(
	w http.ResponseWriter,
	req *http.Request,
	cmd *command.Forward,
	upstream *command.Upstream) error {

	// Rewrites the request: adds headers, changes urls etc.
	outReq := rewriteRequest(req, cmd, upstream)

	// Forward the reuest and mirror the response
	upstream.Metrics.Requests.Mark(1)
	startts := time.Now()
	res, err := p.httpTransport.RoundTrip(outReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	upstream.Metrics.Latency.Update(time.Since(startts))

	// In some cases upstreams may return special error codes that indicate that instead
	// of proxying the response of the upstream to the client we should initiate a failover
	if cmd.Failover != nil && len(cmd.Failover.Codes) != 0 {
		for _, code := range cmd.Failover.Codes {
			if res.StatusCode == code {
				upstream.Metrics.Failovers.Mark(1)
				glog.Errorf("Upstream %s initiated failover with status code %d", upstream, code)
				return fmt.Errorf("Upstream %s initiated failover with status code %d", upstream, code)
			}
		}
	}

	netutils.CopyHeaders(w.Header(), res.Header)
	w.WriteHeader(res.StatusCode)
	upstream.Metrics.Http.MarkResponseCode(res.StatusCode)
	io.Copy(w, res.Body)
	return nil
}

var vulcanHostname, _ = os.Hostname()

const TRUST_FORWARD_HEADER = false

// This function alters the original request - adds/removes headers, removes hop headers,
// changes the request path.
func rewriteRequest(req *http.Request, cmd *command.Forward, upstream *command.Upstream) *http.Request {
	outReq := new(http.Request)
	*outReq = *req // includes shallow copies of maps, but we handle this below

	outReq.URL.Scheme = upstream.Scheme
	outReq.URL.Host = fmt.Sprintf("%s:%d", upstream.Host, upstream.Port)
	if len(cmd.RewritePath) != 0 {
		outReq.URL.Path = cmd.RewritePath
	}

	outReq.URL.RawQuery = req.URL.RawQuery

	outReq.Proto = "HTTP/1.1"
	outReq.ProtoMajor = 1
	outReq.ProtoMinor = 1
	outReq.Close = false

	glog.Infof("Proxying request to: %v", outReq)

	outReq.Header = make(http.Header)
	netutils.CopyHeaders(outReq.Header, req.Header)

	if clientIP, _, err := net.SplitHostPort(req.RemoteAddr); err == nil {
		// TODO(pquerna): configure this?  Not all backends properly parse the header..
		if TRUST_FORWARD_HEADER {
			if prior, ok := outReq.Header["X-Forwarded-For"]; ok {
				clientIP = strings.Join(prior, ", ") + ", " + clientIP
			}
		}
		outReq.Header.Set("X-Forwarded-For", clientIP)
	}

	if req.TLS != nil {
		outReq.Header.Set("X-Forwarded-Proto", "https")
	} else {
		outReq.Header.Set("X-Forwarded-Proto", "http")
	}

	if req.Host != "" {
		outReq.Header.Set("X-Forwarded-Host", req.Host)
	}

	outReq.Header.Set("X-Forwarded-Server", vulcanHostname)

	if len(cmd.RemoveHeaders) != 0 {
		netutils.RemoveHeaders(cmd.RemoveHeaders, outReq.Header)
	}

	// Add generic instructions headers to the request
	if len(cmd.AddHeaders) != 0 {
		glog.Info("Proxying instructions headers:", cmd.AddHeaders)
		netutils.CopyHeaders(outReq.Header, cmd.AddHeaders)
	}

	// Remove hop-by-hop headers to the backend.  Especially
	// important is "Connection" because we want a persistent
	// connection, regardless of what the client sent to us.
	netutils.RemoveHeaders(hopHeaders, outReq.Header)
	return outReq
}

// Helper function to reply with http errors
func (p *ReverseProxy) replyError(err error, w http.ResponseWriter, req *http.Request) {
	httpResponse, err := p.controller.ConvertError(req, err)
	if err != nil {
		glog.Errorf("Error converter failed: %s", err)
		httpResponse = netutils.NewHttpError(http.StatusInternalServerError)
	}
	// Discard the request body, so that clients can actually receive the response
	// Otherwise they can only see lost connection
	// TODO: actually check this
	io.Copy(ioutil.Discard, req.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpResponse.StatusCode)
	w.Write(httpResponse.Body)
}

// Helper function to reply with a response specified in the reply command
func (p *ReverseProxy) replyCommand(cmd *command.Reply, w http.ResponseWriter, req *http.Request) {
	// Discard the request body, so that clients can actually receive the response
	// Otherwise they can only see lost connection.
	// TODO: actually check this in tests
	io.Copy(ioutil.Discard, req.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(cmd.Code)
	body, err := json.Marshal(cmd.Body)
	if err != nil {
		glog.Errorf("Failed to serialize body: %s", err)
		body = []byte("Internal system error")
	}
	w.Write(body)
}

func validateProxySettings(s *ProxySettings) (*ProxySettings, error) {
	if s == nil {
		return nil, fmt.Errorf("Provide proxy settings")
	}
	if s.Controller == nil {
		return nil, fmt.Errorf("Controller can not be nil")
	}
	if s.ThrottlerBackend == nil {
		return nil, fmt.Errorf("Backend can not be nil")
	}
	if s.LoadBalancer == nil {
		return nil, fmt.Errorf("Load balancer can not be nil")
	}
	if s.HttpReadTimeout == time.Duration(0) {
		s.HttpReadTimeout = DefaultHttpReadTimeout
	}
	if s.HttpReadTimeout == time.Duration(0) {
		s.HttpDialTimeout = DefaultHttpDialTimeout
	}
	return s, nil
}
