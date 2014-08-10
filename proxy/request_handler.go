package proxy

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/cloudfoundry/gorouter/access_log"
	"github.com/cloudfoundry/gorouter/common"
	router_http "github.com/cloudfoundry/gorouter/common/http"
	"github.com/cloudfoundry/gorouter/route"
	steno "github.com/cloudfoundry/gosteno"
)

type RequestHandler struct {
	logger    *steno.Logger
	reporter  ProxyReporter
	logrecord *access_log.AccessLogRecord

	request  *http.Request
	response http.ResponseWriter
}

func NewRequestHandler(request *http.Request, response http.ResponseWriter, r ProxyReporter,
	alr *access_log.AccessLogRecord) RequestHandler {
	return RequestHandler{
		logger:    createLogger(request),
		reporter:  r,
		logrecord: alr,

		request:  request,
		response: response,
	}
}

func createLogger(request *http.Request) *steno.Logger {
	logger := steno.NewLogger("router.proxy.request-handler")

	logger.Set("RemoteAddr", request.RemoteAddr)
	logger.Set("Host", request.Host)
	logger.Set("Path", request.URL.Path)
	logger.Set("X-Forwarded-For", request.Header["X-Forwarded-For"])
	logger.Set("X-Forwarded-Proto", request.Header["X-Forwarded-Proto"])

	return logger
}

func (h *RequestHandler) Logger() *steno.Logger {
	return h.logger
}

func (h *RequestHandler) HandleHeartbeat() {
	h.logrecord.StatusCode = http.StatusOK
	h.response.WriteHeader(http.StatusOK)
	h.response.Write([]byte("ok\n"))
	h.request.Close = true
}

func (h *RequestHandler) HandleUnsupportedProtocol() {
	// must be hijacked, otherwise no response is sent back
	conn, buf, err := h.hijack()
	if err != nil {
		h.writeStatus(http.StatusBadRequest, "Unsupported protocol")
		return
	}

	h.logrecord.StatusCode = http.StatusBadRequest
	fmt.Fprintf(buf, "HTTP/1.0 400 Bad Request\r\n\r\n")
	buf.Flush()
	conn.Close()
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

func (h *RequestHandler) HandleTcpRequest(iter route.EndpointIterator) {
	h.logger.Set("Upgrade", "tcp")

	err := h.serveTcp(iter)
	if err != nil {
		h.writeStatus(http.StatusBadRequest, "TCP forwarding to endpoint failed.")
	}
}

func (h *RequestHandler) HandleWebSocketRequest(iter route.EndpointIterator) {
	h.logger.Set("Upgrade", "websocket")

	err := h.serveWebSocket(iter)
	if err != nil {
		h.writeStatus(http.StatusBadRequest, "WebSocket request to endpoint failed.")
	}
}

func (h *RequestHandler) writeStatus(code int, message string) {
	body := fmt.Sprintf("%d %s: %s", code, http.StatusText(code), message)

	h.logger.Warn(body)
	h.logrecord.StatusCode = code

	http.Error(h.response, body, code)
	if code > 299 {
		h.response.Header().Del("Connection")
	}
}

func (h *RequestHandler) serveTcp(iter route.EndpointIterator) error {
	var err error
	var connection net.Conn

	client, _, err := h.hijack()
	if err != nil {
		return err
	}

	defer func() {
		client.Close()
		if connection != nil {
			connection.Close()
		}
	}()

	retry := 0
	for {
		endpoint := iter.Next()
		if endpoint == nil {
			h.reporter.CaptureBadGateway(h.request)
			err = noEndpointsAvailable
			h.HandleBadGateway(err)
			return err
		}

		connection, err = net.DialTimeout("tcp", endpoint.CanonicalAddr(), 5*time.Second)
		if err == nil {
			break
		}

		iter.EndpointFailed()

		h.logger.Set("Error", err.Error())
		h.logger.Warn("proxy.tcp.failed")

		retry++
		if retry == retries {
			return err
		}
	}

	if connection != nil {
		forwardIO(client, connection)
	}

	return nil
}

func (h *RequestHandler) serveWebSocket(iter route.EndpointIterator) error {
	var err error
	var connection net.Conn

	client, _, err := h.hijack()
	if err != nil {
		return err
	}

	defer func() {
		client.Close()
		if connection != nil {
			connection.Close()
		}
	}()

	retry := 0
	for {
		endpoint := iter.Next()
		if endpoint == nil {
			h.reporter.CaptureBadGateway(h.request)
			err = noEndpointsAvailable
			h.HandleBadGateway(err)
			return err
		}

		connection, err = net.DialTimeout("tcp", endpoint.CanonicalAddr(), 5*time.Second)
		if err == nil {
			h.setupRequest(endpoint)
			break
		}

		iter.EndpointFailed()

		h.logger.Set("Error", err.Error())
		h.logger.Warn("proxy.websocket.failed")

		retry++
		if retry == retries {
			return err
		}
	}

	if connection != nil {
		err = h.request.Write(connection)
		if err != nil {
			return err
		}

		forwardIO(client, connection)
	}
	return nil
}

func (h *RequestHandler) setupRequest(endpoint *route.Endpoint) {
	h.setRequestURL(endpoint.CanonicalAddr())
	h.setRequestXForwardedFor()
	setRequestXRequestStart(h.request)
	setRequestXVcapRequestId(h.request, h.logger)
}

func (h *RequestHandler) setRequestURL(addr string) {
	h.request.URL.Scheme = "http"
	h.request.URL.Host = addr
}

func (h *RequestHandler) setRequestXForwardedFor() {
	if clientIP, _, err := net.SplitHostPort(h.request.RemoteAddr); err == nil {
		// If we aren't the first proxy retain prior
		// X-Forwarded-For information as a comma+space
		// separated list and fold multiple headers into one.
		if prior, ok := h.request.Header["X-Forwarded-For"]; ok {
			clientIP = strings.Join(prior, ", ") + ", " + clientIP
		}
		h.request.Header.Set("X-Forwarded-For", clientIP)
	}
}

func setRequestXRequestStart(request *http.Request) {
	if _, ok := request.Header[http.CanonicalHeaderKey("X-Request-Start")]; !ok {
		request.Header.Set("X-Request-Start", strconv.FormatInt(time.Now().UnixNano()/1e6, 10))
	}
}

func setRequestXVcapRequestId(request *http.Request, logger *steno.Logger) {
	uuid, err := common.GenerateUUID()
	if err == nil {
		request.Header.Set(router_http.VcapRequestIdHeader, uuid)
		if logger != nil {
			logger.Set(router_http.VcapRequestIdHeader, uuid)
		}
	}
}

func setRequestXCfInstanceId(request *http.Request, endpoint *route.Endpoint) {
	value := endpoint.PrivateInstanceId
	if value == "" {
		value = endpoint.CanonicalAddr()
	}

	request.Header.Set(router_http.CfInstanceIdHeader, value)
}

func (h *RequestHandler) hijack() (client net.Conn, io *bufio.ReadWriter, err error) {
	hijacker, ok := h.response.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("response writer cannot hijack")
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
