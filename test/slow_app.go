package test

import (
	"io"
	"net/http"
	"time"

	"github.com/cloudfoundry/gorouter/route"
	"github.com/cloudfoundry/yagnats"
)

func NewSlowApp(urls []route.Uri, rPort uint16, mbusClient yagnats.NATSClient, delay time.Duration) *TestApp {
	app := NewTestApp(urls, rPort, mbusClient, nil)

	app.AddHandler("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		io.WriteString(w, "Hello, world")
	})

	app.AddHandler("/hello", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.(http.Flusher).Flush()
		time.Sleep(delay)
		io.WriteString(w, "Hello, world")
	})

	return app
}
