package dropsonde_test

import (
	"errors"
	"github.com/cloudfoundry-incubator/dropsonde"
	"github.com/cloudfoundry-incubator/dropsonde/emitter/fake"
	"github.com/cloudfoundry-incubator/dropsonde/events"
	"github.com/cloudfoundry-incubator/dropsonde/factories"
	uuid "github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net/http"
)

type FakeRoundTripper struct {
	FakeError error
}

func (frt *FakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{StatusCode: 123, ContentLength: 1234}, frt.FakeError
}

var _ = Describe("InstrumentedRoundTripper", func() {
	var fakeRoundTripper *FakeRoundTripper
	var rt http.RoundTripper
	var req *http.Request
	var fakeEmitter *fake.FakeEventEmitter

	var origin = "testRoundtripper/42"

	BeforeEach(func() {
		var err error
		fakeEmitter = fake.NewFakeEventEmitter(origin)

		fakeRoundTripper = new(FakeRoundTripper)
		rt = dropsonde.InstrumentedRoundTripper(fakeRoundTripper, fakeEmitter)

		req, err = http.NewRequest("GET", "http://foo.example.com/", nil)
		Expect(err).ToNot(HaveOccurred())
		req.RemoteAddr = "127.0.0.1"
		req.Header.Set("User-Agent", "our-testing-client")

	})

	Describe("request ID", func() {
		It("should generate a new request ID", func() {
			rt.RoundTrip(req)
			Expect(req.Header.Get("X-CF-RequestID")).ToNot(BeEmpty())
		})

		Context("if request ID can't be generated", func() {
			BeforeEach(func() {
				dropsonde.GenerateUuid = func() (u *uuid.UUID, err error) {
					return nil, errors.New("test error")
				}
			})
			AfterEach(func() {
				dropsonde.GenerateUuid = uuid.NewV4
			})

			It("defaults to an empty request ID", func() {
				rt.RoundTrip(req)
				Expect(req.Header.Get("X-CF-RequestID")).To(Equal("00000000-0000-0000-0000-000000000000"))
			})
		})
	})

	Context("event emission", func() {
		It("should emit a start event", func() {
			rt.RoundTrip(req)
			Expect(fakeEmitter.Messages[0].Event).To(BeAssignableToTypeOf(new(events.HttpStart)))
			Expect(fakeEmitter.Messages[0].Origin).To(Equal("testRoundtripper/42"))
		})

		Context("if request ID already exists", func() {
			var existingRequestId *uuid.UUID

			BeforeEach(func() {
				existingRequestId, _ = uuid.NewV4()
				req.Header.Set("X-CF-RequestID", existingRequestId.String())
			})

			It("should emit the existing request ID as the parent request ID", func() {
				rt.RoundTrip(req)
				startEvent := fakeEmitter.Messages[0].Event.(*events.HttpStart)
				Expect(startEvent.GetParentRequestId()).To(Equal(factories.NewUUID(existingRequestId)))
			})
		})

		Context("if round tripper returns an error", func() {
			It("should emit a stop event with blank response fields", func() {
				fakeRoundTripper.FakeError = errors.New("fakeEmitter error")
				rt.RoundTrip(req)

				Expect(fakeEmitter.Messages[1].Event).To(BeAssignableToTypeOf(new(events.HttpStop)))

				stopEvent := fakeEmitter.Messages[1].Event.(*events.HttpStop)
				Expect(stopEvent.GetStatusCode()).To(BeNumerically("==", 0))
				Expect(stopEvent.GetContentLength()).To(BeNumerically("==", 0))
			})
		})

		Context("if round tripper does not return an error", func() {
			It("should emit a stop event with the round tripper's response", func() {
				rt.RoundTrip(req)

				Expect(fakeEmitter.Messages[1].Event).To(BeAssignableToTypeOf(new(events.HttpStop)))

				stopEvent := fakeEmitter.Messages[1].Event.(*events.HttpStop)
				Expect(stopEvent.GetStatusCode()).To(BeNumerically("==", 123))
				Expect(stopEvent.GetContentLength()).To(BeNumerically("==", 1234))
			})
		})
	})

})
