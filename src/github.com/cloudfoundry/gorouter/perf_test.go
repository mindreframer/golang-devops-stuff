package main_test

import (
	"github.com/cloudfoundry/gorouter/access_log"
	"github.com/cloudfoundry/gorouter/config"
	"github.com/cloudfoundry/gorouter/proxy"
	"github.com/cloudfoundry/gorouter/registry"
	"github.com/cloudfoundry/gorouter/route"
	"github.com/cloudfoundry/gorouter/varz"
	"github.com/cloudfoundry/yagnats/fakeyagnats"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"strconv"
)

var _ = Describe("AccessLogRecord", func() {
	Measure("Register", func(b Benchmarker) {
		c := config.DefaultConfig()
		mbus := fakeyagnats.New()
		r := registry.NewRouteRegistry(c, mbus)

		accesslog, err := access_log.CreateRunningAccessLogger(c)
		Ω(err).ToNot(HaveOccurred())

		proxy.NewProxy(proxy.ProxyArgs{
			EndpointTimeout: c.EndpointTimeout,
			Ip:              c.Ip,
			TraceKey:        c.TraceKey,
			Registry:        r,
			Reporter:        varz.NewVarz(r),
			AccessLogger:    accesslog,
		})

		b.Time("RegisterTime", func() {
			for i := 0; i < 1000; i++ {
				str := strconv.Itoa(i)
				r.Register(
					route.Uri("bench.vcap.me."+str),
					route.NewEndpoint("", "localhost", uint16(i), "", nil),
				)
			}
		})
	}, 10)

})
