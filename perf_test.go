package router

import (
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	"strconv"
	"testing"

	"github.com/cloudfoundry/gorouter/config"
	"github.com/cloudfoundry/gorouter/proxy"
	"github.com/cloudfoundry/gorouter/registry"
	"github.com/cloudfoundry/gorouter/route"
	"github.com/cloudfoundry/gorouter/varz"
)

const (
	Host = "1.2.3.4"
	Port = 1234
)

func BenchmarkRegister(b *testing.B) {
	c := config.DefaultConfig()
	mbus := fakeyagnats.New()
	r := registry.NewRegistry(c, mbus)
	p := proxy.NewProxy(c, r, varz.NewVarz(r))

	for i := 0; i < b.N; i++ {
		str := strconv.Itoa(i)

		p.Register(
			route.Uri("bench.vcap.me."+str),
			&route.Endpoint{
				Host: "localhost",
				Port: uint16(i),
			},
		)
	}
}
