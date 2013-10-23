package errplane

import (
	"fmt"
	. "launchpad.net/gocheck"
	"net"
	"time"
)

type ErrplaneAggregatorApiSuite struct{}

var _ = Suite(&ErrplaneAggregatorApiSuite{})

var (
	udpListener *net.UDPConn
	udpRecorder *UdpRequestRecorder
)

type UdpRequestRecorder struct {
	requests []string
}

func (self *UdpRequestRecorder) recordRequest(conn net.Conn) {
}

func (s *ErrplaneAggregatorApiSuite) SetUpTest(c *C) {
	udpRecorder.requests = nil
}

func (s *ErrplaneAggregatorApiSuite) SetUpSuite(c *C) {
	var err error
	addr, err := net.ResolveUDPAddr("udp4", "")
	c.Assert(err, IsNil)
	udpListener, err = net.ListenUDP("udp4", addr)
	c.Assert(err, IsNil)
	udpRecorder = new(UdpRequestRecorder)
	go func() {
		for {
			buffer := make([]byte, 1024)
			n, _, _, _, err := udpListener.ReadMsgUDP(buffer, nil)
			if err != nil || n <= 0 {
				break
			}
			udpRecorder.requests = append(udpRecorder.requests, string(buffer[:n]))
		}
	}()
	currentTime = time.Now()
}

func (s *ErrplaneAggregatorApiSuite) TearDownSuite(c *C) {
	udpListener.Close()
}

func (s *ErrplaneAggregatorApiSuite) TestDoesNotOverrideUdpAddr(c *C) {
	ep := newTestClient("app4you2love", "staging", "some_key")
	ep.SetUdpAddr(udpListener.LocalAddr().(*net.UDPAddr).String())
	ep.SetHttpHost("localhost") // there was a bug where this call will override the udp addr that the user sets
	c.Assert(ep, NotNil)

	err := ep.ReportUDP("some_metric", 123.4, "doesn't send empty points", Dimensions{
		"foo": "bar",
	})
	c.Assert(err, IsNil)
	ep.Close()

	time.Sleep(200 * time.Millisecond)

	c.Assert(udpRecorder.requests, HasLen, 1)
	expected := fmt.Sprintf(`{"d":"app4you2lovestaging","a":"some_key","o":"r","w":[{"n":"some_metric","p":[{"v":123.4,"c":"doesn't send empty points","d":{"foo":"bar"}}]}]}`)
	c.Assert(udpRecorder.requests, Contains, expected)
}

// the purpose of this test is to make sure that we don't send empty arrays
// to the backend which we were doing.
func (s *ErrplaneAggregatorApiSuite) TestDoesNotSendEmptyPoints(c *C) {
	ep := newTestClient("app4you2love", "staging", "some_key")
	ep.SetUdpAddr(udpListener.LocalAddr().(*net.UDPAddr).String())
	c.Assert(ep, NotNil)

	err := ep.ReportUDP("some_metric", 123.4, "doesn't send empty points", Dimensions{
		"foo": "bar",
	})
	c.Assert(err, IsNil)
	ep.Close()

	time.Sleep(200 * time.Millisecond)

	c.Assert(udpRecorder.requests, HasLen, 1)
	expected := fmt.Sprintf(`{"d":"app4you2lovestaging","a":"some_key","o":"r","w":[{"n":"some_metric","p":[{"v":123.4,"c":"doesn't send empty points","d":{"foo":"bar"}}]}]}`)
	c.Assert(udpRecorder.requests, Contains, expected)
}

func (s *ErrplaneAggregatorApiSuite) TestApi(c *C) {
	ep := newTestClient("app4you2love", "staging", "some_key")
	ep.SetUdpAddr(udpListener.LocalAddr().(*net.UDPAddr).String())
	c.Assert(ep, NotNil)

	err := ep.ReportUDP("some_metric", 123.4, "some_context", Dimensions{
		"foo": "bar",
	})
	c.Assert(err, IsNil)
	time.Sleep(200 * time.Millisecond)
	err = ep.Sum("some_metric", 10, "some_context", Dimensions{
		"foo": "bar",
	})
	c.Assert(err, IsNil)
	time.Sleep(200 * time.Millisecond)
	err = ep.Aggregate("some_metric", 234.5, "some_context", Dimensions{
		"foo": "bar",
	})
	c.Assert(err, IsNil)
	ep.Close()

	time.Sleep(200 * time.Millisecond)

	c.Assert(udpRecorder.requests, HasLen, 3)
	expected := fmt.Sprintf(`{"d":"app4you2lovestaging","a":"some_key","o":"r","w":[{"n":"some_metric","p":[{"v":123.4,"c":"some_context","d":{"foo":"bar"}}]}]}`)
	c.Assert(udpRecorder.requests, Contains, expected)
	expected = fmt.Sprintf(`{"d":"app4you2lovestaging","a":"some_key","o":"t","w":[{"n":"some_metric","p":[{"v":234.5,"c":"some_context","d":{"foo":"bar"}}]}]}`)
	c.Assert(udpRecorder.requests, Contains, expected)
	expected = fmt.Sprintf(`{"d":"app4you2lovestaging","a":"some_key","o":"c","w":[{"n":"some_metric","p":[{"v":10,"c":"some_context","d":{"foo":"bar"}}]}]}`)
	c.Assert(udpRecorder.requests, Contains, expected)
}
