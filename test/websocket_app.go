package test

import (
	"net/http"
	"time"

	"github.com/onsi/ginkgo"

	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/gorouter/route"
	"github.com/cloudfoundry/gorouter/test_util"
	"github.com/cloudfoundry/yagnats"
)

func NewWebSocketApp(urls []route.Uri, rPort uint16, mbusClient yagnats.NATSClient, delay time.Duration) *TestApp {
	app := NewTestApp(urls, rPort, mbusClient, nil)
	app.AddHandler("/", func(w http.ResponseWriter, r *http.Request) {
		defer ginkgo.GinkgoRecover()

		Ω(r.Header.Get("Upgrade")).Should(Equal("websocket"))
		Ω(r.Header.Get("Connection")).Should(Equal("upgrade"))

		conn, _, err := w.(http.Hijacker).Hijack()
		x := test_util.NewHttpConn(conn)

		resp := test_util.NewResponse(http.StatusSwitchingProtocols)
		resp.Header.Set("Upgrade", "websocket")
		resp.Header.Set("Connection", "upgrade")

		time.Sleep(delay)

		x.WriteResponse(resp)
		Ω(err).ShouldNot(HaveOccurred())

		x.CheckLine("hello from client")
		x.WriteLine("hello from server")
	})

	return app
}
