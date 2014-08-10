package router_test

import (
	"github.com/cloudfoundry/gorouter/access_log"
	vcap "github.com/cloudfoundry/gorouter/common"
	cfg "github.com/cloudfoundry/gorouter/config"
	"github.com/cloudfoundry/gorouter/proxy"
	rregistry "github.com/cloudfoundry/gorouter/registry"
	"github.com/cloudfoundry/gorouter/route"
	. "github.com/cloudfoundry/gorouter/router"
	"github.com/cloudfoundry/gorouter/test"
	"github.com/cloudfoundry/gorouter/test_util"
	vvarz "github.com/cloudfoundry/gorouter/varz"
	"github.com/cloudfoundry/gunk/natsrunner"
	"github.com/cloudfoundry/yagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net/http"
	"time"
)

var _ = Describe("Router", func() {
	var natsRunner *natsrunner.NATSRunner
	var config *cfg.Config

	var mbusClient *yagnats.Client
	var registry *rregistry.RouteRegistry
	var varz vvarz.Varz
	var router *Router
	var natsPort uint16

	BeforeEach(func() {
		natsPort = test_util.NextAvailPort()
		natsRunner = natsrunner.NewNATSRunner(int(natsPort))
		natsRunner.Start()

		proxyPort := test_util.NextAvailPort()
		statusPort := test_util.NextAvailPort()

		config = test_util.SpecConfig(natsPort, statusPort, proxyPort)
		config.EndpointTimeout = 5 * time.Second

		mbusClient = natsRunner.MessageBus.(*yagnats.Client)
		registry = rregistry.NewRouteRegistry(config, mbusClient)
		varz = vvarz.NewVarz(registry)
		logcounter := vcap.NewLogCounter()
		proxy := proxy.NewProxy(proxy.ProxyArgs{
			EndpointTimeout: config.EndpointTimeout,
			Ip:              config.Ip,
			TraceKey:        config.TraceKey,
			Registry:        registry,
			Reporter:        varz,
			AccessLogger:    &access_log.NullAccessLogger{},
		})
		r, err := NewRouter(config, proxy, mbusClient, registry, varz, logcounter)
		Ω(err).ShouldNot(HaveOccurred())
		router = r
		r.Run()
	})

	AfterEach(func() {
		if natsRunner != nil {
			natsRunner.Stop()
		}

		if router != nil {
			router.Stop()
		}
	})

	Context("Drain", func() {
		It("waits until the last request completes", func() {
			app := test.NewTestApp([]route.Uri{"drain.vcap.me"}, config.Port, mbusClient, nil)

			blocker := make(chan bool)
			resultCh := make(chan bool, 2)
			app.AddHandler("/", func(w http.ResponseWriter, r *http.Request) {
				blocker <- true

				_, err := ioutil.ReadAll(r.Body)
				defer r.Body.Close()
				Ω(err).ShouldNot(HaveOccurred())

				<-blocker

				w.WriteHeader(http.StatusNoContent)
			})
			app.Listen()
			Ω(waitAppRegistered(registry, app, time.Second*5)).To(BeTrue())

			go func() {
				defer GinkgoRecover()
				req, err := http.NewRequest("GET", app.Endpoint(), nil)
				Ω(err).ShouldNot(HaveOccurred())

				client := http.Client{}
				resp, err := client.Do(req)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(resp).ShouldNot(BeNil())
				defer resp.Body.Close()
				_, err = ioutil.ReadAll(resp.Body)
				Ω(err).ShouldNot(HaveOccurred())
				resultCh <- false
			}()

			<-blocker

			go func() {
				defer GinkgoRecover()
				err := router.Drain(2 * time.Second)
				Ω(err).ShouldNot(HaveOccurred())
				resultCh <- true
			}()

			Consistently(resultCh).ShouldNot(Receive())

			blocker <- false

			var result bool
			Eventually(resultCh).Should(Receive(&result))
			Ω(result).To(BeTrue())
		})

		It("times out if it takes too long", func() {
			app := test.NewTestApp([]route.Uri{"draintimeout.vcap.me"}, config.Port, mbusClient, nil)

			blocker := make(chan bool)
			resultCh := make(chan error, 2)
			app.AddHandler("/", func(w http.ResponseWriter, r *http.Request) {
				blocker <- true

				_, err := ioutil.ReadAll(r.Body)
				defer r.Body.Close()
				Ω(err).ShouldNot(HaveOccurred())

				time.Sleep(1 * time.Second)
			})
			app.Listen()
			Ω(waitAppRegistered(registry, app, time.Second*5)).To(BeTrue())

			go func() {
				defer GinkgoRecover()
				req, err := http.NewRequest("GET", app.Endpoint(), nil)
				Ω(err).ShouldNot(HaveOccurred())

				client := http.Client{}
				resp, err := client.Do(req)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(resp).ShouldNot(BeNil())
				defer resp.Body.Close()
			}()

			<-blocker

			go func() {
				defer GinkgoRecover()
				err := router.Drain(500 * time.Millisecond)
				resultCh <- err
			}()

			var result error
			Eventually(resultCh).Should(Receive(&result))
			Ω(result).Should(Equal(DrainTimeout))
		})
	})
})
