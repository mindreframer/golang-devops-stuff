package servicecontrol

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/mailgun/vulcan/instructions"
	"github.com/mailgun/vulcan/loadbalance"
	"github.com/mailgun/vulcan/netutils"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"time"
)

type Client struct {
	// Load balancing algo
	loadBalancer loadbalance.Balancer
	// Control server urls that decide what to do with the request
	controlServers []*instructions.Upstream
	// Client that uses customized transport
	httpClient *http.Client
}

type Settings struct {
	Servers      []string
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	LoadBalancer loadbalance.Balancer
}

// Standard dial and read timeouts, can be overriden when supplying
// proxy settings
const (
	DefaultHttpReadTimeout = time.Duration(10) * time.Second
	DefaultHttpDialTimeout = time.Duration(10) * time.Second
)

func NewClient(s *Settings) (*Client, error) {
	if s.LoadBalancer == nil {
		return nil, fmt.Errorf("Please provide load balancer")
	}
	if s.DialTimeout <= 0 {
		s.DialTimeout = DefaultHttpDialTimeout
	}

	if s.ReadTimeout <= 0 {
		s.ReadTimeout = DefaultHttpReadTimeout
	}

	transport := &http.Transport{
		Dial: func(network, addr string) (net.Conn, error) {
			return net.DialTimeout(network, addr, s.DialTimeout)
		},
		ResponseHeaderTimeout: s.ReadTimeout,
	}

	client := &Client{
		controlServers: make([]*instructions.Upstream, len(s.Servers)),
		loadBalancer:   s.LoadBalancer,
		httpClient: &http.Client{
			Transport: transport,
		},
	}

	for i, str := range s.Servers {
		u, err := netutils.ParseUrl(str)
		if err != nil {
			return nil, err
		}
		client.controlServers[i] = &instructions.Upstream{Url: u}
	}

	return client, nil
}

// Control request is issued by the client
// to a control server asking what do do with the request
// Control server replies with structured reply - ProxyInstructions
// or denies request based on it's internal logic
type ControlRequest struct {
	Username string
	Password string
	Protocol string
	Method   string
	Url      string
	Length   int64
	Ip       string
	Headers  map[string][]string
}

// Issues a request to an routing server. Three outcomes are possible:
//
// * Request failed. In this case general error is returned.
// * Request has been denied by auth server, in this case HttpError is returned
// * Requst has been granted and auth server replied with instructions
//
func (c *Client) GetInstructions(req *http.Request) (*instructions.ProxyInstructions, error) {
	controlRequest, err := controlRequestFromHttp(req)
	if err != nil {
		if _, ok := err.(AuthError); ok {
			glog.Errorf("Failed to create control request: %s", err)
			return nil, netutils.NewHttpError(http.StatusProxyAuthRequired)
		}
		return nil, err
	}

	endpoints := instructions.EndpointsFromUpstreams(c.controlServers)
	for i := 0; i < len(endpoints); i++ {
		pendpoint, err := c.loadBalancer.NextEndpoint(endpoints)
		if err != nil {
			glog.Errorf("Control server %s denied request %s", pendpoint.Id(), err)
			return nil, err
		}

		endpoint := pendpoint.(*instructions.Endpoint)
		instructions, err := c.queryServer(endpoint.Upstream.Url, controlRequest)
		if err != nil {
			// This is http error that we'd like to transfer to the client
			_, isHttp := err.(*netutils.HttpError)
			if isHttp {
				glog.Errorf("Control server %s denied request %s", endpoint.Upstream, err)
				return nil, err
			} else {
				// mark this endpoint as inactive and try another
				endpoint.Active = false
				glog.Errorf("Control server %s failed: %s, try another", endpoint.Upstream, err)
			}
		} else {
			return instructions, err
		}
	}
	glog.Errorf("All control servers failed")
	return nil, fmt.Errorf("All control servers failed.")
}

func (c *Client) queryServer(controlServer *url.URL, controlRequest *ControlRequest) (*instructions.ProxyInstructions, error) {

	query, err := controlRequest.controlQuery(controlServer)
	if err != nil {
		return nil, fmt.Errorf(
			"Failed to create query for controlServer %s, err %s",
			controlServer, err)
	}

	response, err := c.httpClient.Get(query.String())
	if err != nil {
		return nil, fmt.Errorf(
			"Control request failed. Server %s, error: '%s'",
			controlServer, err)
	}

	defer response.Body.Close()
	responseBody, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, fmt.Errorf(
			"Failed to read response from auth server %s error: %s",
			controlServer, err)
	}
	glog.Infof("ControlServer replies: \n-->%s<--\n", responseBody)

	// Control server denied the request, stream this request
	if response.StatusCode >= 300 || response.StatusCode < 200 {
		return nil, &netutils.HttpError{
			StatusCode: response.StatusCode,
			Status:     response.Status,
			Body:       responseBody}
	}

	instructions, err := instructions.ProxyInstructionsFromJson(responseBody)
	if err != nil {
		return nil, fmt.Errorf(
			"Failed to decode auth response %s error: %s",
			responseBody, err)
	}

	return instructions, nil
}

func controlRequestFromHttp(r *http.Request) (*ControlRequest, error) {
	auth, err := netutils.ParseAuthHeader(r.Header.Get("Authorization"))
	if err != nil {
		return nil, AuthError(err.Error())
	}

	request := &ControlRequest{
		Username: auth.Username,
		Password: auth.Password,
		Protocol: r.Proto,
		Method:   r.Method,
		Url:      r.RequestURI,
		Length:   r.ContentLength,
		Headers:  r.Header,
	}

	return request, nil
}

func (r *ControlRequest) controlQuery(controlServer *url.URL) (*url.URL, error) {
	u := netutils.CopyUrl(controlServer)

	encodedHeaders, err := json.Marshal(r.Headers)
	if err != nil {
		return nil, err
	}

	parameters := url.Values{}
	parameters.Add("username", r.Username)
	parameters.Add("password", r.Password)
	parameters.Add("protocol", r.Protocol)
	parameters.Add("method", r.Method)
	parameters.Add("url", r.Url)
	parameters.Add("length", fmt.Sprintf("%d", r.Length))
	parameters.Add("headers", string(encodedHeaders))

	u.RawQuery = parameters.Encode()

	return u, nil
}

// We somehow failed to authenticate the request
type AuthError string

func (f AuthError) Error() string {
	return string(f)
}
