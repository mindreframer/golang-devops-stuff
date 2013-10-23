package router

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os/exec"
	"regexp"
	"strings"
	"time"

	mbus "github.com/cloudfoundry/go_cfmessagebus"
	. "launchpad.net/gocheck"

	"github.com/cloudfoundry/gorouter/common"
	"github.com/cloudfoundry/gorouter/config"
	"github.com/cloudfoundry/gorouter/proxy"
	"github.com/cloudfoundry/gorouter/registry"
	"github.com/cloudfoundry/gorouter/route"
	"github.com/cloudfoundry/gorouter/test"
)

type RouterSuite struct {
	Config        *config.Config
	natsServerCmd *exec.Cmd
	mbusClient    mbus.MessageBus
	router        *Router
	natsPort      uint16
}

var _ = Suite(&RouterSuite{})

func (s *RouterSuite) SetUpSuite(c *C) {
	s.natsPort = nextAvailPort()

	s.natsServerCmd = StartNats(int(s.natsPort))

	proxyPort := nextAvailPort()
	statusPort := nextAvailPort()

	s.Config = SpecConfig(s.natsPort, statusPort, proxyPort)

	s.router = NewRouter(s.Config)
	go s.router.Run()

	<-s.WaitUntilNatsIsUp()
	s.mbusClient = s.router.mbusClient
}

func (s *RouterSuite) TearDownSuite(c *C) {
	StopNats(s.natsServerCmd)
}

func (s *RouterSuite) TestRouterGreets(c *C) {
	response := make(chan []byte)

	s.mbusClient.Request("router.greet", []byte{}, func(payload []byte) {
		response <- payload
	})

	select {
	case msg := <-response:
		c.Assert(string(msg), Matches, ".*\"minimumRegisterIntervalInSeconds\":5.*")
	case <-time.After(500 * time.Millisecond):
		c.Error("Did not see a response to router.greet!")
	}
}

func (s *RouterSuite) TestDiscover(c *C) {
	// Test if router responses to discover message
	sig := make(chan common.VcapComponent)

	// Since the form of uptime is xxd:xxh:xxm:xxs, we should make
	// sure that router has run at least for one second
	time.Sleep(time.Second)

	s.mbusClient.Request("vcap.component.discover", []byte{}, func(payload []byte) {
		var component common.VcapComponent
		_ = json.Unmarshal(payload, &component)
		sig <- component
	})

	cc := <-sig

	var emptyTime time.Time
	var emptyDuration common.Duration

	c.Check(cc.Type, Equals, "Router")
	c.Check(cc.Index, Equals, uint(2))
	c.Check(cc.UUID, Not(Equals), "")
	c.Check(cc.Start, Not(Equals), emptyTime)
	c.Check(cc.Uptime, Not(Equals), emptyDuration)

	verify_var_z(cc.Host, cc.Credentials[0], cc.Credentials[1], c)
	verify_health_z(cc.Host, s.router.registry, c)
}

func (s *RouterSuite) waitMsgReceived(a *test.TestApp, r bool, t time.Duration) bool {
	i := time.Millisecond * 50
	m := int(t / i)

	for j := 0; j < m; j++ {
		received := true
		for _, v := range a.Urls() {
			_, ok := s.router.registry.Lookup(v)
			if ok != r {
				received = false
				break
			}
		}
		if received {
			return true
		}
		time.Sleep(i)
	}

	return false
}

func (s *RouterSuite) waitAppRegistered(app *test.TestApp, timeout time.Duration) bool {
	return s.waitMsgReceived(app, true, timeout)
}

func (s *RouterSuite) waitAppUnregistered(app *test.TestApp, timeout time.Duration) bool {
	return s.waitMsgReceived(app, false, timeout)
}

func (s *RouterSuite) TestRegisterUnregister(c *C) {
	app := test.NewGreetApp([]route.Uri{"test.vcap.me"}, s.Config.Port, s.mbusClient, nil)
	app.Listen()
	c.Assert(s.waitAppRegistered(app, time.Second*5), Equals, true)

	app.VerifyAppStatus(200, c)

	app.Unregister()
	c.Assert(s.waitAppUnregistered(app, time.Second*5), Equals, true)
	app.VerifyAppStatus(404, c)
}

func (s *RouterSuite) TestRegistryLastUpdatedVarz(c *C) {
	initialUpdateTime := f(s.readVarz(), "ms_since_last_registry_update").(float64)

	app1 := test.NewGreetApp([]route.Uri{"test1.vcap.me"}, s.Config.Port, s.mbusClient, nil)
	app1.Listen()
	c.Assert(s.waitAppRegistered(app1, time.Second*5), Equals, true)

	// varz time should be different
	updateTime := f(s.readVarz(), "ms_since_last_registry_update").(float64)

	c.Assert(updateTime < initialUpdateTime, Equals, true)
}

func (s *RouterSuite) readVarz() map[string]interface{} {
	x, err := s.router.varz.MarshalJSON()
	if err != nil {
		panic(err)
	}

	y := make(map[string]interface{})
	err = json.Unmarshal(x, &y)
	if err != nil {
		panic(err)
	}

	return y
}

func f(x interface{}, s ...string) interface{} {
	var ok bool

	for _, y := range s {
		z := x.(map[string]interface{})
		x, ok = z[y]
		if !ok {
			panic(fmt.Sprintf("no key: %s", s))
		}
	}

	return x
}

func (s *RouterSuite) TestVarz(c *C) {
	app := test.NewGreetApp([]route.Uri{"count.vcap.me"}, s.Config.Port, s.mbusClient, map[string]string{"framework": "rails"})
	app.Listen()

	c.Assert(s.waitAppRegistered(app, time.Millisecond*500), Equals, true)
	// Send seed request
	sendRequests(c, "count.vcap.me", s.Config.Port, 1)
	vA := s.readVarz()

	// Send requests
	sendRequests(c, "count.vcap.me", s.Config.Port, 100)
	vB := s.readVarz()

	// Verify varz update
	RequestsA := int(f(vA, "requests").(float64))
	RequestsB := int(f(vB, "requests").(float64))
	allRequests := RequestsB - RequestsA
	c.Check(allRequests, Equals, 100)

	Responses2xxA := int(f(vA, "responses_2xx").(float64))
	Responses2xxB := int(f(vB, "responses_2xx").(float64))
	allResponses2xx := Responses2xxB - Responses2xxA
	c.Check(allResponses2xx, Equals, 100)

	app.Unregister()
}

func (s *RouterSuite) TestStickySession(c *C) {
	apps := make([]*test.TestApp, 10)
	for i := range apps {
		apps[i] = test.NewStickyApp([]route.Uri{"sticky.vcap.me"}, s.Config.Port, s.mbusClient, nil)
		apps[i].Listen()
	}

	for _, app := range apps {
		c.Assert(s.waitAppRegistered(app, time.Millisecond*500), Equals, true)
	}
	sessionCookie, vcapCookie, port1 := getSessionAndAppPort("sticky.vcap.me", s.Config.Port, c)
	port2 := getAppPortWithSticky("sticky.vcap.me", s.Config.Port, sessionCookie, vcapCookie, c)

	c.Check(port1, Equals, port2)
	c.Check(vcapCookie.Path, Equals, "/")

	for _, app := range apps {
		app.Unregister()
	}
}

func timeoutDialler() func(net, addr string) (c net.Conn, err error) {
	return func(netw, addr string) (net.Conn, error) {
		c, err := net.Dial(netw, addr)
		c.SetDeadline(time.Now().Add(2 * time.Second))
		return c, err
	}
}

func verify_health_z(host string, registry *registry.Registry, c *C) {
	var req *http.Request
	var resp *http.Response
	var err error
	path := "/healthz"

	req, _ = http.NewRequest("GET", "http://"+host+path, nil)
	bytes := verify_success(req, c)
	c.Check(err, IsNil)
	c.Check(string(bytes), Equals, "ok")

	// Check that healthz does not reply during deadlock
	registry.Lock()
	defer registry.Unlock()

	httpClient := http.Client{
		Transport: &http.Transport{
			Dial: timeoutDialler(),
		},
	}

	req, err = http.NewRequest("GET", "http://"+host+path, nil)
	resp, err = httpClient.Do(req)

	c.Assert(err, Not(IsNil))
	match, _ := regexp.Match("i/o timeout", []byte(err.Error()))
	c.Assert(match, Equals, true)
	c.Check(resp, IsNil)

}

func verify_var_z(host, user, pass string, c *C) {
	var client http.Client
	var req *http.Request
	var resp *http.Response
	var err error
	path := "/varz"

	// Request without username:password should be rejected
	req, _ = http.NewRequest("GET", "http://"+host+path, nil)
	resp, err = client.Do(req)
	c.Check(err, IsNil)
	c.Assert(resp, Not(IsNil))
	c.Check(resp.StatusCode, Equals, 401)

	// varz Basic auth
	req.SetBasicAuth(user, pass)
	bytes := verify_success(req, c)
	varz := make(map[string]interface{})
	json.Unmarshal(bytes, &varz)

	c.Assert(varz["num_cores"], Not(Equals), 0)
	c.Assert(varz["type"], Equals, "Router")
	c.Assert(varz["uuid"], Not(Equals), "")
}

func verify_success(req *http.Request, c *C) []byte {
	var client http.Client
	resp, err := client.Do(req)
	defer resp.Body.Close()

	c.Check(err, IsNil)
	c.Assert(resp, Not(IsNil))
	c.Check(resp.StatusCode, Equals, 200)

	bytes, err := ioutil.ReadAll(resp.Body)
	c.Check(err, IsNil)

	return bytes
}

func (s *RouterSuite) TestRouterRunErrors(c *C) {
	c.Assert(func() { s.router.Run() }, PanicMatches, "net.Listen.*")
}

func (s *RouterSuite) TestProxyPutRequest(c *C) {
	app := test.NewTestApp([]route.Uri{"greet.vcap.me"}, s.Config.Port, s.mbusClient, nil)

	var rr *http.Request
	var msg string
	app.AddHandler("/", func(w http.ResponseWriter, r *http.Request) {
		rr = r
		b, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		msg = string(b)
	})
	app.Listen()
	c.Assert(s.waitAppRegistered(app, time.Second*5), Equals, true)

	url := app.Endpoint()

	buf := bytes.NewBufferString("foobar")
	r, err := http.NewRequest("PUT", url, buf)
	c.Assert(err, IsNil)

	resp, err := http.DefaultClient.Do(r)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, http.StatusOK)

	c.Assert(rr, NotNil)
	c.Assert(rr.Method, Equals, "PUT")
	c.Assert(rr.Proto, Equals, "HTTP/1.1")
	c.Assert(msg, Equals, "foobar")
}

func (s *RouterSuite) TestRouterSendsStartOnConnect(c *C) {
	started := make(chan bool)

	s.router.mbusClient.Subscribe("router.start", func([]byte) {
		started <- true
	})

	StopNats(s.natsServerCmd)
	s.natsServerCmd = StartNats(int(s.natsPort))
	<-s.WaitUntilNatsIsUp()

	select {
	case <-started:
	case <-time.After(500 * time.Millisecond):
		c.Error("Did not receive router.start!")
	}
}

func (s *RouterSuite) WaitUntilNatsIsUp() chan bool {
	natsConnected := make(chan bool, 1)
	go func() {
		for {
			if s.router.mbusClient.Publish("asdf", []byte("data")) == nil {
				break
			}
			time.Sleep(500 * time.Millisecond)
		}
		natsConnected <- true
	}()
	return natsConnected
}

func (s *RouterSuite) Test100ContinueRequest(c *C) {
	app := test.NewTestApp([]route.Uri{"foo.vcap.me"}, s.Config.Port, s.mbusClient, nil)
	rCh := make(chan *http.Request)
	app.AddHandler("/", func(w http.ResponseWriter, r *http.Request) {
		_, err := ioutil.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
		}
		rCh <- r
	})

	<-s.WaitUntilNatsIsUp()

	app.Listen()
	c.Assert(s.waitAppRegistered(app, time.Second*5), Equals, true)

	host := fmt.Sprintf("foo.vcap.me:%d", s.Config.Port)
	conn, err := net.Dial("tcp", host)
	c.Assert(err, IsNil)
	defer conn.Close()

	fmt.Fprintf(conn, "POST / HTTP/1.1\r\n"+
		"Host: %s\r\n"+
		"Connection: close\r\n"+
		"Content-Length: 1\r\n"+
		"Expect: 100-continue\r\n"+
		"\r\n", host)

	fmt.Fprintf(conn, "a")

	buf := bufio.NewReader(conn)
	line, err := buf.ReadString('\n')
	c.Assert(err, IsNil)
	c.Assert(strings.Contains(line, "100 Continue"), Equals, true)

	rr := <-rCh
	c.Assert(rr, NotNil)
	c.Assert(rr.Header.Get("Expect"), Equals, "")
}

func sendRequests(c *C, url string, rPort uint16, times int) {
	uri := fmt.Sprintf("http://%s:%d", url, rPort)

	for i := 0; i < times; i++ {
		r, err := http.Get(uri)
		if err != nil {
			panic(err)
		}

		c.Check(r.StatusCode, Equals, http.StatusOK)
	}
}

func getSessionAndAppPort(url string, rPort uint16, c *C) (*http.Cookie, *http.Cookie, string) {
	var client http.Client
	var req *http.Request
	var resp *http.Response
	var err error
	var port []byte

	uri := fmt.Sprintf("http://%s:%d/sticky", url, rPort)

	req, err = http.NewRequest("GET", uri, nil)
	c.Assert(err, IsNil)

	resp, err = client.Do(req)
	c.Assert(err, IsNil)

	port, err = ioutil.ReadAll(resp.Body)
	c.Assert(err, IsNil)

	var sessionCookie, vcapCookie *http.Cookie
	for _, cookie := range resp.Cookies() {
		if cookie.Name == proxy.StickyCookieKey {
			sessionCookie = cookie
		} else if cookie.Name == proxy.VcapCookieId {
			vcapCookie = cookie
		}
	}

	return sessionCookie, vcapCookie, string(port)
}

func getAppPortWithSticky(url string, rPort uint16, sessionCookie, vcapCookie *http.Cookie, c *C) string {
	var client http.Client
	var req *http.Request
	var resp *http.Response
	var err error
	var port []byte

	uri := fmt.Sprintf("http://%s:%d/sticky", url, rPort)

	req, err = http.NewRequest("GET", uri, nil)
	c.Assert(err, IsNil)

	req.AddCookie(sessionCookie)
	req.AddCookie(vcapCookie)

	resp, err = client.Do(req)
	c.Assert(err, IsNil)

	port, err = ioutil.ReadAll(resp.Body)

	return string(port)
}

func (s *RouterSuite) TestInfoApi(c *C) {
	var client http.Client
	var req *http.Request
	var resp *http.Response
	var err error

	<-s.WaitUntilNatsIsUp()
	s.mbusClient.Publish("router.register", []byte(`{"dea":"dea1","app":"app1","uris":["test.com"],"host":"1.2.3.4","port":1234,"tags":{},"private_instance_id":"private_instance_id"}`))
	time.Sleep(250 * time.Millisecond)

	host := fmt.Sprintf("http://%s:%d/routes", s.Config.Ip, s.Config.Status.Port)

	req, err = http.NewRequest("GET", host, nil)
	req.SetBasicAuth("user", "pass")

	resp, err = client.Do(req)
	c.Assert(err, IsNil)
	c.Assert(resp, Not(IsNil))
	c.Check(resp.StatusCode, Equals, 200)

	body, err := ioutil.ReadAll(resp.Body)
	defer resp.Body.Close()
	c.Assert(err, IsNil)
	c.Check(string(body), Matches, ".*1\\.2\\.3\\.4:1234.*\n")
}

func (s *RouterSuite) TestRouterTerminatesLongRequests(c *C) {
	app := test.NewSlowApp(
		[]route.Uri{"slow-app.vcap.me"},
		s.Config.Port,
		s.mbusClient,
		10*time.Second,
	)

	app.Listen()

	uri := fmt.Sprintf("http://slow-app.vcap.me:%d", s.Config.Port)
	resp, err := http.Get(uri)
	c.Assert(err, IsNil)
	c.Assert(resp.StatusCode, Equals, 502)
}
