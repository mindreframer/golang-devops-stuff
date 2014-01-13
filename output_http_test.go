package gor

import (
	"io"
	"net/http"
	"sync"
	"testing"
)

func TestHTTPOutput(t *testing.T) {
	startHTTP := func(addr string, cb func(*http.Request)) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cb(r)
		})

		go http.ListenAndServe(addr, handler)
	}

	wg := new(sync.WaitGroup)
	quit := make(chan int)

	input := NewTestInput()

	headers := HTTPHeaders{HTTPHeader{"User-Agent", "Gor"}}
	methods := HTTPMethods{"GET", "PUT", "POST"}
	output := NewHTTPOutput("127.0.0.1:50003", headers, methods, "")

	startHTTP("127.0.0.1:50003", func(req *http.Request) {
		if req.Header.Get("User-Agent") != "Gor" {
			t.Error("Wrong header")
		}

		if req.Method == "OPTIONS" {
			t.Error("Wrong method")
		}

		wg.Done()
	})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}

	go Start(quit)

	for i := 0; i < 100; i++ {
		wg.Add(2)
		input.EmitPOST()
		input.EmitOPTIONS()
		input.EmitGET()
	}

	wg.Wait()

	close(quit)
}
