package factories_test

import (
	uuid "github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"code.google.com/p/gogoprotobuf/proto"
	"github.com/cloudfoundry-incubator/dropsonde/events"
	"github.com/cloudfoundry-incubator/dropsonde/factories"
	"net/http"
)

var _ = Describe("HTTP event creation", func() {
	var applicationId *uuid.UUID
	var requestId *uuid.UUID
	var req *http.Request

	BeforeEach(func() {
		applicationId, _ = uuid.NewV4()
		requestId, _ = uuid.NewV4()
		req, _ = http.NewRequest("GET", "http://foo.example.com/", nil)

		req.RemoteAddr = "127.0.0.1"
		req.Header.Set("User-Agent", "our-testing-client")
	})

	Describe("NewHttpStart", func() {

		Context("without an application ID or instanceIndex", func() {

			It("should set appropriate fields", func() {
				expectedStartEvent := &events.HttpStart{
					RequestId:     factories.NewUUID(requestId),
					PeerType:      events.PeerType_Server.Enum(),
					Method:        events.HttpStart_GET.Enum(),
					Uri:           proto.String("foo.example.com/"),
					RemoteAddress: proto.String("127.0.0.1"),
					UserAgent:     proto.String("our-testing-client"),
				}

				startEvent := factories.NewHttpStart(req, events.PeerType_Server, requestId)

				Expect(startEvent.GetTimestamp()).ToNot(BeZero())
				startEvent.Timestamp = nil

				Expect(startEvent).To(Equal(expectedStartEvent))
			})
		})

		Context("with an application ID", func() {
			It("should include it in the start event", func() {
				applicationId, _ := uuid.NewV4()
				req.Header.Set("X-CF-ApplicationID", applicationId.String())

				startEvent := factories.NewHttpStart(req, events.PeerType_Server, requestId)

				Expect(startEvent.GetApplicationId()).To(Equal(factories.NewUUID(applicationId)))
			})
		})

		Context("with an application instance index", func() {
			It("should include it in the start event", func() {
				req.Header.Set("X-CF-InstanceIndex", "1")

				startEvent := factories.NewHttpStart(req, events.PeerType_Server, requestId)

				Expect(startEvent.GetInstanceIndex()).To(BeNumerically("==", 1))
			})
		})

		Context("with an application instance ID", func() {
			It("should include it in the start event", func() {
				req.Header.Set("X-CF-InstanceID", "fake-id")

				startEvent := factories.NewHttpStart(req, events.PeerType_Server, requestId)

				Expect(startEvent.GetInstanceId()).To(Equal("fake-id"))
			})
		})
	})

	Describe("NewHttpStop", func() {
		BeforeEach(func() {
			req.Header.Set("X-CF-ApplicationID", applicationId.String())
		})

		It("should set appropriate fields", func() {
			expectedStopEvent := &events.HttpStop{
				ApplicationId: factories.NewUUID(applicationId),
				RequestId:     factories.NewUUID(requestId),
				Uri:           proto.String("foo.example.com/"),
				PeerType:      events.PeerType_Server.Enum(),
				StatusCode:    proto.Int32(200),
				ContentLength: proto.Int64(3),
			}

			stopEvent := factories.NewHttpStop(req, 200, 3, events.PeerType_Server, requestId)

			Expect(stopEvent.GetTimestamp()).ToNot(BeZero())
			stopEvent.Timestamp = nil

			Expect(stopEvent).To(Equal(expectedStopEvent))
		})
	})

	Describe("StringFromUUID", func() {
		It("returns a string for a UUID", func() {
			id := factories.NewUUID(requestId)
			Expect(factories.StringFromUUID(id)).To(Equal(requestId.String()))
		})
	})
})
