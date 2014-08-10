package varz_test

import (
	"github.com/cloudfoundry/gorouter/config"
	"github.com/cloudfoundry/gorouter/registry"
	"github.com/cloudfoundry/gorouter/route"
	. "github.com/cloudfoundry/gorouter/varz"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

var _ = Describe("Varz", func() {
	var Varz Varz
	var Registry *registry.RouteRegistry

	BeforeEach(func() {
		Registry = registry.NewRouteRegistry(config.DefaultConfig(), fakeyagnats.New())
		Varz = NewVarz(Registry)
	})

	It("contains the following items", func() {
		v := Varz

		members := []string{
			"responses_2xx",
			"responses_3xx",
			"responses_4xx",
			"responses_5xx",
			"responses_xxx",
			"latency",
			"rate",
			"tags",
			"urls",
			"droplets",
			"requests",
			"bad_requests",
			"bad_gateways",
			"requests_per_sec",
			"top10_app_requests",
			"ms_since_last_registry_update",
		}

		b, e := json.Marshal(v)
		Ω(e).ShouldNot(HaveOccurred())

		d := make(map[string]interface{})
		e = json.Unmarshal(b, &d)
		Ω(e).ShouldNot(HaveOccurred())

		for _, k := range members {
			_, ok := d[k]
			Ω(ok).Should(BeTrue(), k)
		}
	})

	It("reports seconds since last registry update", func() {
		Registry.Register("foo", &route.Endpoint{})

		time.Sleep(10 * time.Millisecond)

		timeSince := findValue(Varz, "ms_since_last_registry_update").(float64)
		Ω(timeSince).Should(BeNumerically("<", 1000))
		Ω(timeSince).Should(BeNumerically(">=", 10))
	})

	It("has urls", func() {
		Ω(findValue(Varz, "urls")).To(Equal(float64(0)))

		var fooReg = route.NewEndpoint("12345", "192.168.1.1", 1234, "", map[string]string{})

		// Add a route
		Registry.Register("foo.vcap.me", fooReg)
		Registry.Register("fooo.vcap.me", fooReg)

		Ω(findValue(Varz, "urls")).To(Equal(float64(2)))
	})

	It("updates bad requests", func() {
		r := http.Request{}

		Varz.CaptureBadRequest(&r)
		Ω(findValue(Varz, "bad_requests")).To(Equal(float64(1)))

		Varz.CaptureBadRequest(&r)
		Ω(findValue(Varz, "bad_requests")).To(Equal(float64(2)))
	})

	It("updates bad gateways", func() {
		r := &http.Request{}

		Varz.CaptureBadGateway(r)
		Ω(findValue(Varz, "bad_gateways")).To(Equal(float64(1)))

		Varz.CaptureBadGateway(r)
		Ω(findValue(Varz, "bad_gateways")).To(Equal(float64(2)))
	})

	It("updates requests", func() {
		b := &route.Endpoint{}
		r := http.Request{}

		Varz.CaptureRoutingRequest(b, &r)
		Ω(findValue(Varz, "requests")).To(Equal(float64(1)))

		Varz.CaptureRoutingRequest(b, &r)
		Ω(findValue(Varz, "requests")).To(Equal(float64(2)))
	})

	It("updates requests with tags", func() {
		b1 := &route.Endpoint{
			Tags: map[string]string{
				"component": "cc",
			},
		}

		b2 := &route.Endpoint{
			Tags: map[string]string{
				"component": "cc",
			},
		}

		r1 := http.Request{}
		r2 := http.Request{}

		Varz.CaptureRoutingRequest(b1, &r1)
		Varz.CaptureRoutingRequest(b2, &r2)

		Ω(findValue(Varz, "tags", "component", "cc", "requests")).To(Equal(float64(2)))
	})

	It("updates responses", func() {
		var b *route.Endpoint = &route.Endpoint{}
		var t time.Time
		var d time.Duration

		r1 := &http.Response{
			StatusCode: http.StatusOK,
		}

		r2 := &http.Response{
			StatusCode: http.StatusNotFound,
		}

		Varz.CaptureRoutingResponse(b, r1, t, d)
		Varz.CaptureRoutingResponse(b, r2, t, d)
		Varz.CaptureRoutingResponse(b, r2, t, d)

		Ω(findValue(Varz, "responses_2xx")).To(Equal(float64(1)))
		Ω(findValue(Varz, "responses_4xx")).To(Equal(float64(2)))
	})

	It("update responses with tags", func() {
		var t time.Time
		var d time.Duration

		b1 := &route.Endpoint{
			Tags: map[string]string{
				"component": "cc",
			},
		}

		b2 := &route.Endpoint{
			Tags: map[string]string{
				"component": "cc",
			},
		}

		r1 := &http.Response{
			StatusCode: http.StatusOK,
		}

		r2 := &http.Response{
			StatusCode: http.StatusNotFound,
		}

		Varz.CaptureRoutingResponse(b1, r1, t, d)
		Varz.CaptureRoutingResponse(b2, r2, t, d)
		Varz.CaptureRoutingResponse(b2, r2, t, d)

		Ω(findValue(Varz, "tags", "component", "cc", "responses_2xx")).To(Equal(float64(1)))
		Ω(findValue(Varz, "tags", "component", "cc", "responses_4xx")).To(Equal(float64(2)))
	})

	It("updates response latency", func() {
		var routeEndpoint *route.Endpoint = &route.Endpoint{}
		var startedAt = time.Now()
		var duration = 1 * time.Millisecond

		response := &http.Response{
			StatusCode: http.StatusOK,
		}

		Varz.CaptureRoutingResponse(routeEndpoint, response, startedAt, duration)

		Ω(findValue(Varz, "latency", "50").(float64)).To(Equal(float64(duration) / float64(time.Second)))
		Ω(findValue(Varz, "latency", "75").(float64)).To(Equal(float64(duration) / float64(time.Second)))
		Ω(findValue(Varz, "latency", "90").(float64)).To(Equal(float64(duration) / float64(time.Second)))
		Ω(findValue(Varz, "latency", "95").(float64)).To(Equal(float64(duration) / float64(time.Second)))
		Ω(findValue(Varz, "latency", "99").(float64)).To(Equal(float64(duration) / float64(time.Second)))
	})
})

// Extract value using key(s) from JSON data
// For example, when extracting value from
//       {
//         "foo": { "bar" : 1 },
//         "foobar": 2,
//        }
// findValue(Varz,"foo", "bar") returns 1
// findValue(Varz,"foobar") returns 2
func findValue(varz Varz, x ...string) interface{} {
	var z interface{}
	var ok bool

	b, err := json.Marshal(varz)
	Ω(err).ShouldNot(HaveOccurred())

	y := make(map[string]interface{})
	err = json.Unmarshal(b, &y)
	Ω(err).ShouldNot(HaveOccurred())
	z = y

	for _, e := range x {
		u := z.(map[string]interface{})
		z, ok = u[e]
		Ω(ok).Should(BeTrue(), fmt.Sprintf("no key: %s", e))
	}

	return z
}
