package proxy

import (
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	steno "github.com/cloudfoundry/gosteno"

	"github.com/cloudfoundry/gorouter/config"
	"github.com/cloudfoundry/gorouter/registry"
	"github.com/cloudfoundry/gorouter/route"
	"github.com/cloudfoundry/gorouter/varz"
)

const (
	VcapTraceHeader = "X-Vcap-Trace"

	VcapCookieId    = "__VCAP_ID__"
	StickyCookieKey = "JSESSIONID"
)

type Proxy struct {
	sync.RWMutex
	*steno.Logger
	*config.Config
	*registry.Registry
	varz.Varz
	*AccessLogger
	*http.Transport
}

func NewProxy(c *config.Config, registry *registry.Registry, v varz.Varz) *Proxy {
	p := &Proxy{
		Config:    c,
		Logger:    steno.NewLogger("router.proxy"),
		Registry:  registry,
		Varz:      v,
		Transport: &http.Transport{ResponseHeaderTimeout: c.EndpointTimeout},
	}

	loggregatorUrl := c.LoggregatorConfig.Url
	loggregatorSharedSecret := c.LoggregatorConfig.SharedSecret
	if c.AccessLog != "" || loggregatorUrl != "" {
		f, err := os.OpenFile(c.AccessLog, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil && c.AccessLog != "" {
			panic(err)
		}

		p.AccessLogger = NewAccessLogger(f, loggregatorUrl, loggregatorSharedSecret, c.Index)
		go p.AccessLogger.Run()
	}

	return p
}

func hostWithoutPort(req *http.Request) string {
	host := req.Host

	// Remove :<port>
	pos := strings.Index(host, ":")
	if pos >= 0 {
		host = host[0:pos]
	}

	return host
}

func (proxy *Proxy) Lookup(request *http.Request) (*route.Endpoint, bool) {
	uri := route.Uri(hostWithoutPort(request))

	// Try choosing a backend using sticky session
	if _, err := request.Cookie(StickyCookieKey); err == nil {
		if sticky, err := request.Cookie(VcapCookieId); err == nil {
			routeEndpoint, ok := proxy.Registry.LookupByPrivateInstanceId(uri, sticky.Value)
			if ok {
				return routeEndpoint, ok
			}
		}
	}

	// Choose backend using host alone
	return proxy.Registry.Lookup(uri)
}

func (proxy *Proxy) ServeHTTP(responseWriter http.ResponseWriter, request *http.Request) {
	startedAt := time.Now()

	handler := NewRequestHandler(request, responseWriter)

	accessLog := AccessLogRecord{
		Request:   request,
		StartedAt: startedAt,
	}

	if !isProtocolSupported(request) {
		handler.HandleUnsupportedProtocol()
		return
	}

	if isLoadBalancerHeartbeat(request) {
		handler.HandleHeartbeat()
		return
	}

	routeEndpoint, found := proxy.Lookup(request)
	if !found {
		proxy.Varz.CaptureBadRequest(request)
		handler.HandleMissingRoute()
		return
	}

	handler.logger.Set("RouteEndpoint", routeEndpoint.ToLogData())

	accessLog.RouteEndpoint = routeEndpoint

	proxy.Varz.CaptureRoutingRequest(routeEndpoint, handler.request)

	if isTcpUpgrade(request) {
		handler.HandleTcpRequest(routeEndpoint)
		return
	}

	if isWebSocketUpgrade(request) {
		handler.HandleWebSocketRequest(routeEndpoint)
		return
	}

	endpointResponse, err := handler.HandleHttpRequest(proxy.Transport, routeEndpoint)

	latency := time.Since(startedAt)

	proxy.Registry.CaptureRoutingRequest(routeEndpoint, startedAt)
	proxy.Varz.CaptureRoutingResponse(routeEndpoint, endpointResponse, latency)

	if err != nil {
		proxy.Varz.CaptureBadGateway(request)
		handler.HandleBadGateway(err)
		return
	}

	accessLog.FirstByteAt = time.Now()
	accessLog.Response = endpointResponse

	if proxy.Config.TraceKey != "" && request.Header.Get(VcapTraceHeader) == proxy.Config.TraceKey {
		handler.SetTraceHeaders(proxy.Config.Ip, routeEndpoint.CanonicalAddr())
	}

	bytesSent := handler.WriteResponse(endpointResponse)

	accessLog.FinishedAt = time.Now()
	accessLog.BodyBytesSent = bytesSent

	if proxy.AccessLogger != nil {
		proxy.AccessLogger.Log(accessLog)
	}
}

func isProtocolSupported(request *http.Request) bool {
	return request.ProtoMajor == 1 && (request.ProtoMinor == 0 || request.ProtoMinor == 1)
}

func isLoadBalancerHeartbeat(request *http.Request) bool {
	return request.UserAgent() == "HTTP-Monitor/1.1"
}

func isWebSocketUpgrade(request *http.Request) bool {
	// websocket should be case insensitive per RFC6455 4.2.1
	return strings.ToLower(upgradeHeader(request)) == "websocket"
}

func isTcpUpgrade(request *http.Request) bool {
	return upgradeHeader(request) == "tcp"
}

func upgradeHeader(request *http.Request) string {
	// upgrade should be case insensitive per RFC6455 4.2.1
	if strings.ToLower(request.Header.Get("Connection")) == "upgrade" {
		return request.Header.Get("Upgrade")
	} else {
		return ""
	}
}
