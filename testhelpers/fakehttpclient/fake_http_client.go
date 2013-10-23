package fakehttpclient

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type FakeHttpClient struct {
	Requests []*Request
}

type Request struct {
	*http.Request
	Callback func(*http.Response, error)
}

func NewFakeHttpClient() *FakeHttpClient {
	client := &FakeHttpClient{}
	client.Reset()
	return client
}

func (client *FakeHttpClient) Reset() {
	client.Requests = make([]*Request, 0)
}

func (client *FakeHttpClient) LastRequest() *Request {
	return client.Requests[len(client.Requests)-1]
}

func (client *FakeHttpClient) Do(req *http.Request, callback func(*http.Response, error)) {
	client.Requests = append(client.Requests, &Request{
		Request:  req,
		Callback: callback,
	})
}

func (request *Request) RespondWithStatus(statusCode int) {
	request.Respond(statusCode, []byte(""), nil)
}

func (request *Request) RespondWithError(err error) {
	request.Callback(nil, err)
}

func (request *Request) Succeed(body []byte) {
	request.Respond(http.StatusOK, body, nil)
}

func (request *Request) Respond(statusCode int, body []byte, err error) {
	reader := strings.NewReader(string(body))
	response := &http.Response{
		Status:     fmt.Sprintf("%d %s", statusCode, http.StatusText(statusCode)),
		StatusCode: statusCode,

		ContentLength: int64(reader.Len()),
		Body:          ioutil.NopCloser(reader),
	}

	request.Callback(response, err)
}
