package test

import (
	mbus "github.com/cloudfoundry/go_cfmessagebus"
	"io"
	"net/http"

	"github.com/cloudfoundry/gorouter/route"
)

func NewGreetApp(urls []route.Uri, rPort uint16, mbusClient mbus.MessageBus, tags map[string]string) *TestApp {
	app := NewTestApp(urls, rPort, mbusClient, tags)
	app.AddHandler("/", greetHandler)

	return app
}

func greetHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Hello, world")
}
