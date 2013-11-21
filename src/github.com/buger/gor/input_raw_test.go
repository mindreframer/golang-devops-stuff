package gor

import (
	"io"
	"net/http"
	"sync"
	"testing"
)

func TestRAWInput(t *testing.T) {
	startHTTP := func(addr string, cb func(*http.Request)) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cb(r)
		})

		go http.ListenAndServe(addr, handler)
	}

	wg := new(sync.WaitGroup)
	quit := make(chan int)

	input := NewRAWInput("127.0.0.1:50004")
	output := NewTestOutput(func(data []byte) {
		wg.Done()
	})

	startHTTP("127.0.0.1:50004", func(req *http.Request) {})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}

	go Start(quit)

	wg.Add(100)
	for i := 0; i < 100; i++ {
		res, _ := http.Get("http://127.0.0.1:50004")
		res.Body.Close()
	}

	wg.Wait()

	close(quit)
}
