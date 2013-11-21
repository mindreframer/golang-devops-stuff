package test

import (
	"github.com/cloudfoundry/yagnats"
	"io"
	"net/http"

	"github.com/cloudfoundry/gorouter/route"
)

func NewGreetApp(urls []route.Uri, rPort uint16, mbusClient yagnats.NATSClient, tags map[string]string) *TestApp {
	app := NewTestApp(urls, rPort, mbusClient, tags)
	app.AddHandler("/", greetHandler)

	return app
}

func greetHandler(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Hello, world")
}
