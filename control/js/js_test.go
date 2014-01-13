package js

import (
	"encoding/json"
	"fmt"
	"github.com/golang/glog"
	"github.com/mailgun/vulcan/client"
	"github.com/mailgun/vulcan/command"
	"github.com/mailgun/vulcan/netutils"
	. "launchpad.net/gocheck"
	"net/http"
)

type JsSuite struct {
	Client *client.RecordingClient
}

var _ = Suite(&JsSuite{})

func (s *JsSuite) executeCode(request *http.Request, code string) (interface{}, error) {
	s.Client = &client.RecordingClient{}
	controller := &JsController{
		CodeGetter: NewStringGetter(code),
		Client:     s.Client,
	}
	return controller.GetInstructions(request)
}

func (s *JsSuite) convertError(request *http.Request, code string, err error) (*netutils.HttpError, error) {
	s.Client = &client.RecordingClient{}
	controller := &JsController{
		CodeGetter: NewStringGetter(code),
		Client:     s.Client,
	}
	return controller.ConvertError(request, err)
}

func (s *JsSuite) expectReply(r *http.Request, code string) *command.Reply {
	replyI, err := s.executeCode(r, code)
	if err != nil {
		panic(err)
	}
	reply, ok := replyI.(*command.Reply)
	if !ok {
		panic(fmt.Errorf("Expected Reply, got: %T", replyI))
	}
	return reply
}

func (s *JsSuite) expectForward(r *http.Request, code string) *command.Forward {
	forwardI, err := s.executeCode(r, code)
	if err != nil {
		panic(err)
	}
	forward, ok := forwardI.(*command.Forward)
	if !ok {
		panic(fmt.Errorf("Expected Forward, got: %T", forwardI))
	}
	return forward
}

func (s *JsSuite) TestReturnReply(c *C) {
	reply := s.expectReply(
		NewTestRequest("GET", "http://localhost", nil),
		`function handle(request){return {code: 200, body: "OK"}}`,
	)
	c.Assert(reply.Code, Equals, 200)
	c.Assert(reply.Body, DeepEquals, "OK")
}

// Make sure that logging does not break the flow no matter what
func (s *JsSuite) TestLogging(c *C) {
	reply := s.expectReply(
		NewTestRequest("GET", "http://localhost", nil),
		`function handle(request){
             info(-1)
             error([])
             info("Hello")
             error("Hello")
             info("Hello %s")
             error("Hello %d")
             info("Hello %s", 1)
             error("Hello %d", "haha")
             info("Hello %d", 1)
             error("Hello %s", "haha")
             info("Hello %v", {omg: "hi"})
             error("Hello %v", {hehe: 1})
             return {code: 200, body: "OK"}
         }`)
	c.Assert(reply.Code, Equals, 200)
	c.Assert(reply.Body, DeepEquals, "OK")
}

func (s *JsSuite) TestReturnForward(c *C) {
	forward := s.expectForward(
		NewTestRequest("GET", "http://localhost", nil),
		`function handle(request){return {upstreams: ["http://localhost:5000"]}}`,
	)
	c.Assert(
		forward.Upstreams,
		DeepEquals,
		[]*command.Upstream{NewTestUpstream("http://localhost:5000")})
}

// Make sure that failures do not cause any panic
func (s *JsSuite) TestHandlerFailures(c *C) {
	chunks := []string{
		"hello",
		"function wrong(){",
		"function wrong(){}",
		"function handle(){}",
		"function handle(){return null}",
		"function handle(){return -1}",
	}
	for _, chunk := range chunks {
		_, err := s.executeCode(NewTestRequest("GET", "http://localhost:5000", nil), chunk)
		glog.Errorf("Error: %s", err)
		c.Assert(err, Not(Equals), nil)
	}
}

func (s *JsSuite) TestConvertError(c *C) {
	elements := []struct {
		Expected *netutils.HttpError
		Code     string
		Error    error
	}{
		{
			Expected: &netutils.HttpError{StatusCode: 500, Body: []byte(`{"error":"Internal Server Error"}`)},
			Code:     `function handleError(request, error){return error}`,
			Error:    fmt.Errorf("Some Error"),
		},
		{
			Expected: &netutils.HttpError{StatusCode: 429, Body: []byte(`{"error":"Too Many Requests", "retry_seconds": 34}`)},
			Code:     `function handleError(request, error){return error}`,
			Error:    &command.RetryError{Seconds: 34},
		},
		{
			Expected: &netutils.HttpError{StatusCode: 502, Body: []byte(`{"error":"Bad Gateway"}`)},
			Code:     `function handleError(request, error){return error}`,
			Error:    &command.AllUpstreamsDownError{},
		},
		// In case of absense of the error handler we return the error code as is
		{
			Expected: &netutils.HttpError{StatusCode: 502, Body: []byte(`{"error":"Bad Gateway"}`)},
			Code:     ``,
			Error:    &command.AllUpstreamsDownError{},
		},
		// We also may transform the error
		{
			Expected: &netutils.HttpError{StatusCode: 506, Body: []byte(`{"error":"Bad Gateway", "hi": "there"}`)},
			Code:     `function handleError(request, error){error.code = 506; error.body["hi"] = "there"; return error;}`,
			Error:    &command.AllUpstreamsDownError{},
		},
	}
	for _, el := range elements {
		out, err := s.convertError(
			NewTestRequest("GET", "http://localhost", nil),
			el.Code,
			el.Error)
		c.Assert(err, Equals, nil)
		c.Assert(out, Not(Equals), nil)
		c.Assert(out.StatusCode, Equals, el.Expected.StatusCode)

		var outBody interface{}
		err = json.Unmarshal(out.Body, &outBody)
		c.Assert(err, Equals, nil)

		var expectedBody interface{}
		err = json.Unmarshal(el.Expected.Body, &expectedBody)
		c.Assert(err, Equals, nil)

		c.Assert(outBody, DeepEquals, expectedBody)
	}
}

// Make sure that failures do not cause any panic
func (s *JsSuite) TestErrorHandlerFailures(c *C) {
	chunks := []string{
		"hello",
		"function wrong(){",
		"function handleError(){}",
		"function handleError(){return null}",
		"function handleError(){return -1}",
	}
	for _, chunk := range chunks {
		_, err := s.convertError(NewTestRequest("GET", "http://localhost:5000", nil), chunk, fmt.Errorf("Some error"))
		if err == nil {
			glog.Errorf("Expected error when executing chunk: %s", chunk)
		}
		c.Assert(err, Not(Equals), nil)
	}
}
