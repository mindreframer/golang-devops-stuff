package httpclient

import (
	"crypto/tls"
	"net"
	"net/http"
	"time"
)

func NewHttpClient(skipSSLVerification bool, timeout time.Duration) HttpClient {
	dialFunc := func(network, addr string) (net.Conn, error) {
		conn, err := net.DialTimeout(network, addr, timeout)
		if err != nil {
			return nil, err
		}
		conn.SetDeadline(time.Now().Add(timeout))
		return conn, err
	}

	transport := &http.Transport{
		Dial: dialFunc,
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: skipSSLVerification,
		},
	}

	return &RealHttpClient{
		client: &http.Client{
			Transport: transport,
		},
	}
}

type RealHttpClient struct {
	client *http.Client
}

func (client *RealHttpClient) Do(req *http.Request, callback func(*http.Response, error)) {
	response, err := client.client.Do(req)
	callback(response, err)
}
