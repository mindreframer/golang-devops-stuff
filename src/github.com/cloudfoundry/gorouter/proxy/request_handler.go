package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/cloudfoundry/gorouter/route"
	steno "github.com/cloudfoundry/gosteno"
)

const (
	VcapBackendHeader     = "X-Vcap-Backend"
	CfRouteEndpointHeader = "X-Cf-RouteEndpoint"
	VcapRouterHeader      = "X-Vcap-Router"
)

type RequestHandler struct {
	logger *steno.Logger

	request  *http.Request
	response http.ResponseWriter

	transport *http.Transport
}

func NewRequestHandler(request *http.Request, response http.ResponseWriter) RequestHandler {
	logger := steno.NewLogger("router.proxy.request-handler")

	logger.Set("RemoteAddr", request.RemoteAddr)
	logger.Set("Host", request.Host)
	logger.Set("Path", request.URL.Path)
	logger.Set("X-Forwarded-For", request.Header["X-Forwarded-For"])
	logger.Set("X-Forwarded-Proto", request.Header["X-Forwarded-Proto"])

	return RequestHandler{
		logger: logger,

		request:  request,
		response: response,
	}
}

func (h *RequestHandler) HandleHeartbeat() {
	h.response.WriteHeader(http.StatusOK)
	h.response.Write([]byte("ok\n"))
}

func (h *RequestHandler) HandleUnsupportedProtocol() {
	client, connection, err := h.hijack()
	if err != nil {
		h.writeStatus(http.StatusBadRequest, "Unsupported protocol.")
		return
	}

	fmt.Fprintf(connection, "HTTP/1.0 400 Bad Request\r\n\r\n")
	connection.Flush()
	client.Close()
}

func (h *RequestHandler) HandleMissingRoute() {
	h.logger.Warnf("proxy.endpoint.not-found")
	h.response.Header().Set("X-Cf-RouterError", "unknown_route")
	message := fmt.Sprintf("Requested route ('%s') does not exist.", h.request.Host)
	h.writeStatus(http.StatusNotFound, message)
}

func (h *RequestHandler) HandleBadGateway(err error) {
	h.logger.Set("Error", err.Error())
	h.logger.Warnf("proxy.endpoint.failed")
	h.response.Header().Set("X-Cf-RouterError", "endpoint_failure")
	h.writeStatus(http.StatusBadGateway, "Registered endpoint failed to handle the request.")
}

func (h *RequestHandler) HandleTcpRequest(endpoint *route.Endpoint) {
	h.logger.Set("Upgrade", "tcp")

	err := h.serveTcp(endpoint)
	if err != nil {
		h.logger.Set("Error", err.Error())
		h.logger.Warn("proxy.tcp.failed")
		h.writeStatus(http.StatusBadRequest, "TCP forwarding to endpoint failed.")
	}
}

func (h *RequestHandler) HandleWebSocketRequest(endpoint *route.Endpoint) {
	h.setupRequest(endpoint)

	h.logger.Set("Upgrade", "websocket")

	err := h.serveWebSocket(endpoint)
	if err != nil {
		h.logger.Set("Error", err.Error())
		h.logger.Warn("proxy.websocket.failed")
		h.writeStatus(http.StatusBadRequest, "WebSocket request to endpoint failed.")
	}
}

func (h *RequestHandler) HandleHttpRequest(transport *http.Transport, endpoint *route.Endpoint) (*http.Response, error) {
	h.transport = transport

	h.setupRequest(endpoint)
	h.setupConnection()

	endpointResponse, err := transport.RoundTrip(h.request)
	if err != nil {
		return endpointResponse, err
	}

	h.forwardResponseHeaders(endpointResponse)

	h.setupStickySession(endpointResponse, endpoint)

	return endpointResponse, err
}

func (h *RequestHandler) SetTraceHeaders(routerIp, addr string) {
	h.response.Header().Set(VcapRouterHeader, routerIp)
	h.response.Header().Set(VcapBackendHeader, addr)
	h.response.Header().Set(CfRouteEndpointHeader, addr)
}

func (h *RequestHandler) WriteResponse(endpointResponse *http.Response) int64 {
	h.response.WriteHeader(endpointResponse.StatusCode)

	bytesSent, err := h.copyToResponse(endpointResponse.Body)
	if err != nil {
		h.logger.Set("Error", err.Error())
		h.logger.Warnf("proxy.response.copy-failed")
	}

	return bytesSent
}

func (h *RequestHandler) copyToResponse(src io.ReadCloser) (int64, error) {
	if src == nil {
		return 0, nil
	}

	var dst io.Writer = h.response

	// Use MaxLatencyFlusher if needed
	if v, ok := h.response.(writeFlusher); ok {
		u := NewMaxLatencyWriter(v, 50*time.Millisecond)
		defer u.Stop()
		dst = u
	}

	copied, err := io.Copy(dst, src)
	if err != nil {
		h.transport.CancelRequest(h.request)
	}

	return copied, err

}

func (h *RequestHandler) setupRequest(endpoint *route.Endpoint) {
	h.setRequestURL(endpoint.CanonicalAddr())
	h.setRequestXForwardedFor()
}

func (h *RequestHandler) setRequestURL(addr string) {
	h.request.URL.Scheme = "http"
	h.request.URL.Host = addr
}

func (h *RequestHandler) setRequestXForwardedFor() {
	if host, _, err := net.SplitHostPort(h.request.RemoteAddr); err == nil {
		// We assume there is a trusted upstream (L7 LB) that properly
		// strips client's XFF header

		// This is sloppy but fine since we don't share this request or
		// headers. Otherwise we should copy the underlying header and
		// append
		xForwardFor := append(h.request.Header["X-Forwarded-For"], host)
		h.request.Header.Set("X-Forwarded-For", strings.Join(xForwardFor, ", "))
	}
}

func (h *RequestHandler) setupConnection() {
	// Use a new connection for every request
	// Keep-alive can be bolted on later, if we want to
	h.request.Close = true
	h.request.Header.Del("Connection")
}

func (h *RequestHandler) serveTcp(endpoint *route.Endpoint) error {
	var err error

	client, _, err := h.hijack()
	if err != nil {
		return err
	}

	connection, err := net.Dial("tcp", endpoint.CanonicalAddr())
	if err != nil {
		return err
	}

	defer client.Close()
	defer connection.Close()

	forwardIO(client, connection)

	return nil
}

func (h *RequestHandler) serveWebSocket(endpoint *route.Endpoint) error {
	var err error

	client, _, err := h.hijack()
	if err != nil {
		return err
	}

	connection, err := net.Dial("tcp", endpoint.CanonicalAddr())
	if err != nil {
		return err
	}

	defer client.Close()
	defer connection.Close()

	err = h.request.Write(connection)
	if err != nil {
		return err
	}

	forwardIO(client, connection)

	return nil
}

func (h *RequestHandler) forwardResponseHeaders(endpointResponse *http.Response) {
	for k, vv := range endpointResponse.Header {
		for _, v := range vv {
			h.response.Header().Add(k, v)
		}
	}
}

func (h *RequestHandler) setupStickySession(endpointResponse *http.Response, endpoint *route.Endpoint) {
	needSticky := false
	for _, v := range endpointResponse.Cookies() {
		if v.Name == StickyCookieKey {
			needSticky = true
			break
		}
	}

	if needSticky && endpoint.PrivateInstanceId != "" {
		cookie := &http.Cookie{
			Name:  VcapCookieId,
			Value: endpoint.PrivateInstanceId,
			Path:  "/",
		}

		http.SetCookie(h.response, cookie)
	}
}

func (h *RequestHandler) writeStatus(code int, message string) {
	body := fmt.Sprintf("%d %s: %s", code, http.StatusText(code), message)

	h.logger.Warn(body)

	http.Error(h.response, body, code)
}

func (h *RequestHandler) hijack() (client net.Conn, io *bufio.ReadWriter, err error) {
	hijacker, ok := h.response.(http.Hijacker)
	if !ok {
		panic("response writer cannot hijack")
	}

	return hijacker.Hijack()
}

func forwardIO(a, b net.Conn) {
	done := make(chan bool, 2)

	copy := func(dst io.Writer, src io.Reader) {
		// don't care about errors here
		io.Copy(dst, src)
		done <- true
	}

	go copy(a, b)
	go copy(b, a)

	<-done
}
