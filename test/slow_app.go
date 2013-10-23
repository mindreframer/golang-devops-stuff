package test

import (
	mbus "github.com/cloudfoundry/go_cfmessagebus"
	"io"
	"net/http"
	"time"

	"github.com/cloudfoundry/gorouter/route"
)

func NewSlowApp(urls []route.Uri, rPort uint16, mbusClient mbus.MessageBus, delay time.Duration) *TestApp {
	app := NewTestApp(urls, rPort, mbusClient, nil)

	app.AddHandler("/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		io.WriteString(w, "Hello, world")
	})

	return app
}
