// Proxy accepts the request, calls the control service for instructions
// And takes actions according to instructions received.
package vulcan

import (
	"bytes"
	"fmt"
	"github.com/golang/glog"
	"github.com/mailgun/vulcan/backend"
	"github.com/mailgun/vulcan/control"
	. "github.com/mailgun/vulcan/instructions"
	"github.com/mailgun/vulcan/loadbalance"
	"github.com/mailgun/vulcan/netutils"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"time"
)

// Defines Reverse proxy runtime settings, what loadbalancing algo to use,
// timeouts, throttling backend.
type ProxySettings struct {
	// Controlller that tells proxy what to do with the request
	// as controller implementation may vary
	Controller control.Controller
	// Any backend that would be used by throttler to keep throttling stats,
	// e.g. MemoryBackend or CassandraBackend
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
	// Controller that decides what to do with the request
	controller control.Controller
	// Filters upstreams based on the throtting data
	throttler *Throttler
	// Sorts upstreams, control servers in accrordance to it's internal
	// algorithm
	loadBalancer loadbalance.Balancer
	// Customized transport with dial and read timeouts set
	httpTransport *http.Transport
	// Client that uses customized transport
	httpClient *http.Client
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

// Creates reverse proxy that acts like http server
func NewReverseProxy(s *ProxySettings) (*ReverseProxy, error) {
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

	p := &ReverseProxy{
		controller:    s.Controller,
		throttler:     NewThrottler(s.ThrottlerBackend),
		loadBalancer:  s.LoadBalancer,
		httpTransport: transport,
		httpClient: &http.Client{
			Transport: transport,
		},
	}
	return p, nil
}

func (p *ReverseProxy) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	glog.Infof("Serving Request %s %s", req.Method, req.RequestURI)

	// Ask controller for instructions
	instructions, err := p.controller.GetInstructions(req)
	if err != nil {
		p.replyError(err, w, req)
		return
	}

	// Get upstreams ready to process the request
	endpoints, err := p.rateLimit(instructions)
	if err != nil {
		p.replyError(err, w, req)
		return
	}

	// Proxy request to the selected upstream
	upstream, err := p.proxyRequest(w, req, instructions, endpoints)
	if err != nil {
		glog.Error("Failed to proxy to the upstreams:", err)
		p.replyError(err, w, req)
		return
	}

	// Update usage stats
	err = p.throttler.updateStats(instructions.Tokens, upstream)
	if err != nil {
		glog.Error("Failed to update stats:", err)
	}
}

func (p *ReverseProxy) rateLimit(instructions *ProxyInstructions) ([]loadbalance.Endpoint, error) {
	// Throttle the requests to find available upstreams
	// We may fall back to all upstreams if throttler is down
	// If there are no available upstreams, we reject the request
	upstreamStats, retrySeconds, err := p.throttler.throttle(instructions)
	if err != nil {
		// throtller is down, we are falling back
		// so we won't loose the request
		glog.Error("Throtter is down, returning all upstreams")
		return EndpointsFromUpstreams(instructions.Upstreams), nil
	} else if retrySeconds > 0 {
		// No available upstreams
		return nil, netutils.TooManyRequestsError(retrySeconds)
	} else {
		return endpointsFromStats(upstreamStats), nil
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

func (p *ReverseProxy) nextEndpoint(endpoints []loadbalance.Endpoint) (*Endpoint, error) {
	// Get first endpoint
	pendpoint, err := p.loadBalancer.NextEndpoint(endpoints)
	if err != nil {
		glog.Errorf("Loadbalancer failure: %s", err)
		return nil, err
	}
	endpoint, ok := pendpoint.(*Endpoint)
	if !ok {
		return nil, fmt.Errorf("Failed to convert types! Unknown type: %v", pendpoint)
	}
	return endpoint, nil
}

func (p *ReverseProxy) proxyRequest(
	w http.ResponseWriter, req *http.Request,
	instructions *ProxyInstructions,
	endpoints []loadbalance.Endpoint) (*Upstream, error) {

	if instructions.Failover == nil || !instructions.Failover.Active {
		endpoint, err := p.nextEndpoint(endpoints)
		if err != nil {
			glog.Errorf("Load Balancer failure: %s", err)
			return nil, err
		}
		glog.Infof("Without failover, proxy to upstream: %s", endpoint.Upstream)
		err = p.proxyToUpstream(w, req, instructions, endpoint.Upstream)
		if err != nil {
			glog.Errorf("Upstream error: %s", err)
			return nil, netutils.NewHttpError(http.StatusBadGateway)
		}
		return endpoint.Upstream, nil
	}

	// We are allowed to fallback in case of upstream failure,
	// so let us record the request body so we can replay
	// it on errors actually
	buffer, err := ioutil.ReadAll(req.Body)
	if err != nil {
		glog.Errorf("Request read error %s", err)
		return nil, netutils.NewHttpError(http.StatusBadRequest)
	}
	reader := &Buffer{bytes.NewReader(buffer)}
	req.Body = reader

	for i := 0; i < len(endpoints); i++ {
		_, err := reader.Seek(0, 0)
		if err != nil {
			return nil, err
		}
		endpoint, err := p.nextEndpoint(endpoints)
		if err != nil {
			glog.Errorf("Load Balancer failure: %s", err)
			return nil, err
		}
		glog.Infof("With failover, proxy to upstream: %s", endpoint.Upstream)
		err = p.proxyToUpstream(w, req, instructions, endpoint.Upstream)
		if err != nil {
			glog.Errorf("Upstream %s error, falling back to another", endpoint.Upstream)
			endpoint.Active = false
		} else {
			return endpoint.Upstream, nil
		}
	}

	glog.Errorf("All upstreams failed!")
	return nil, netutils.NewHttpError(http.StatusBadGateway)
}

func (p *ReverseProxy) proxyToUpstream(
	w http.ResponseWriter,
	req *http.Request,
	instructions *ProxyInstructions,
	upstream *Upstream) error {

	// Rewrites the request: adds headers, changes urls etc.
	outReq := rewriteRequest(req, instructions, upstream)

	// Forward the reuest and mirror the response
	res, err := p.httpTransport.RoundTrip(outReq)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	// In some cases upstreams may return special error codes that indicate that instead
	// of proxying the response of the upstream to the client we should initiate a failover
	if instructions.Failover != nil && len(instructions.Failover.Codes) != 0 {
		for _, code := range instructions.Failover.Codes {
			if res.StatusCode == code {
				glog.Errorf("Upstream %s initiated failover with status code %d", upstream, code)
				return fmt.Errorf("Upstream %s initiated failover with status code %d", upstream, code)
			}
		}
	}

	netutils.CopyHeaders(w.Header(), res.Header)

	w.WriteHeader(res.StatusCode)
	io.Copy(w, res.Body)
	return nil
}

func rewriteRequest(req *http.Request, instructions *ProxyInstructions, upstream *Upstream) *http.Request {
	outReq := new(http.Request)
	*outReq = *req // includes shallow copies of maps, but we handle this below

	outReq.URL.Scheme = upstream.Url.Scheme
	outReq.URL.Host = upstream.Url.Host
	outReq.URL.Path = upstream.Url.Path
	outReq.URL.RawQuery = req.URL.RawQuery

	outReq.Proto = "HTTP/1.1"
	outReq.ProtoMajor = 1
	outReq.ProtoMinor = 1
	outReq.Close = false

	// We copy headers only if we alter the original request
	// headers, otherwise we use the shallow copy
	if len(instructions.Headers) != 0 || len(upstream.Headers) != 0 || netutils.HasHeaders(hopHeaders, req.Header) {
		outReq.Header = make(http.Header)
		netutils.CopyHeaders(outReq.Header, req.Header)
	}

	// Add upstream headers to the request
	if len(upstream.Headers) != 0 {
		glog.Info("Proxying Upstream headers:", upstream.Headers)
		netutils.CopyHeaders(outReq.Header, upstream.Headers)
	}

	// Add generic instructions headers to the request
	if len(instructions.Headers) != 0 {
		glog.Info("Proxying instructions headers:", instructions.Headers)
		netutils.CopyHeaders(outReq.Header, instructions.Headers)
	}

	// Remove hop-by-hop headers to the backend.  Especially
	// important is "Connection" because we want a persistent
	// connection, regardless of what the client sent to us.
	netutils.RemoveHeaders(hopHeaders, outReq.Header)
	return outReq
}

// Helper function to reply with http errors
func (p *ReverseProxy) replyError(err error, w http.ResponseWriter, req *http.Request) {
	httpErr, isHttp := err.(*netutils.HttpError)
	if !isHttp {
		httpErr = netutils.NewHttpError(http.StatusInternalServerError)
	}

	// Discard the request body, so that clients can actually receive the response
	// Otherwise they can only see lost connection
	// TODO: actually check this
	io.Copy(ioutil.Discard, req.Body)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpErr.StatusCode)
	w.Write(httpErr.Body)
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

func endpointsFromStats(upstreamStats []*UpstreamStats) []loadbalance.Endpoint {
	endpoints := make([]loadbalance.Endpoint, len(upstreamStats))
	for i, us := range upstreamStats {
		endpoints[i] = NewEndpoint(us.upstream, !us.ExceededLimits())
	}
	return endpoints
}
