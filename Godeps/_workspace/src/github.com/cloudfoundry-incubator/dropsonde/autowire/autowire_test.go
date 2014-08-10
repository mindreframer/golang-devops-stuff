package autowire_test

import (
	"github.com/cloudfoundry-incubator/dropsonde/autowire"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
	"os"
	"reflect"
)

var _ = Describe("Autowire", func() {
	var oldEnv string

	BeforeEach(func() {
		oldEnv = os.Getenv("DROPSONDE_DESTINATION")
	})

	AfterEach(func() {
		os.Setenv("DROPSONDE_DESTINATION", oldEnv)
	})

	Context("with DROPSONDE_ORIGIN set", func() {
		BeforeEach(func() {
			os.Setenv("DROPSONDE_ORIGIN", "anything")
		})
		Context("with DROPSONDE_DESTINATION missing", func() {
			It("defaults to localhost", func() {
				os.Setenv("DROPSONDE_DESTINATION", "")
				autowire.Initialize()

				Expect(autowire.Destination()).To(Equal("localhost:42420"))
			})
		})

		Context("with DROPSONDE_DESTINATION set", func() {
			It("uses the configured destination", func() {
				os.Setenv("DROPSONDE_DESTINATION", "test")
				autowire.Initialize()

				Expect(autowire.Destination()).To(Equal("test"))
			})
		})
	})
	Context("with DROPSONDE_ORIGIN missing", func() {
		BeforeEach(func() {
			oldEnv = os.Getenv("DROPSONDE_ORIGIN")
		})

		AfterEach(func() {
			os.Setenv("DROPSONDE_ORIGIN", oldEnv)
		})

		It("sets http.DefaultTransport to the non-instrumented default", func() {
			os.Setenv("DROPSONDE_ORIGIN", "")
			autowire.Initialize()

			Expect(reflect.TypeOf(http.DefaultTransport).Elem().Name()).To(Equal("Transport"))
		})

		Describe("InstrumentedHandler", func() {
			It("returns the given Handler with no changes", func() {
				os.Setenv("DROPSONDE_ORIGIN", "")
				fake := FakeHandler{}
				autowire.Initialize()

				Expect(autowire.InstrumentedHandler(fake)).To(Equal(fake))
			})
		})

		Describe("InstrumentedRoundTripper", func() {
			It("returns the given RoundTripper with no changes", func() {
				fake := FakeRoundTripper{}
				os.Setenv("DROPSONDE_ORIGIN", "")
				autowire.Initialize()

				Expect(autowire.InstrumentedRoundTripper(fake)).To(Equal(fake))
			})
		})
	})
})

type FakeHandler struct{}

func (fh FakeHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {}

type FakeRoundTripper struct{}

func (frt FakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, nil
}
