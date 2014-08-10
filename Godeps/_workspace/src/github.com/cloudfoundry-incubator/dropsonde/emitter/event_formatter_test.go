package emitter_test

import (
	"code.google.com/p/gogoprotobuf/proto"
	"github.com/cloudfoundry-incubator/dropsonde/emitter"
	"github.com/cloudfoundry-incubator/dropsonde/events"
	"github.com/cloudfoundry-incubator/dropsonde/factories"
	uuid "github.com/nu7hatch/gouuid"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type unknownEvent struct{}

func (*unknownEvent) ProtoMessage() {}

var _ = Describe("EventFormatter", func() {
	Describe("wrap", func() {
		var origin string

		BeforeEach(func() {
			origin = "testEventFormatter/42"
		})

		It("should work with HttpStart events", func() {
			id, _ := uuid.NewV4()
			testEvent := &events.HttpStart{RequestId: factories.NewUUID(id)}

			envelope, err := emitter.Wrap(testEvent, origin)
			Expect(err).To(BeNil())
			Expect(envelope.GetEventType()).To(Equal(events.Envelope_HttpStart))
			Expect(envelope.GetHttpStart()).To(Equal(testEvent))
		})

		It("should work with HttpStop events", func() {
			id, _ := uuid.NewV4()
			testEvent := &events.HttpStop{RequestId: factories.NewUUID(id)}

			envelope, err := emitter.Wrap(testEvent, origin)
			Expect(err).To(BeNil())
			Expect(envelope.GetEventType()).To(Equal(events.Envelope_HttpStop))
			Expect(envelope.GetHttpStop()).To(Equal(testEvent))
		})

		It("should error with unknown events", func() {
			envelope, err := emitter.Wrap(new(unknownEvent), origin)
			Expect(envelope).To(BeNil())
			Expect(err).ToNot(BeNil())
		})

		It("should work with dropsonde status events", func() {
			statusEvent := &events.Heartbeat{SentCount: proto.Uint64(1), ErrorCount: proto.Uint64(0)}
			envelope, err := emitter.Wrap(statusEvent, origin)
			Expect(err).To(BeNil())
			Expect(envelope.GetEventType()).To(Equal(events.Envelope_Heartbeat))
			Expect(envelope.GetHeartbeat()).To(Equal(statusEvent))
		})

		It("should check that origin is non-empty", func() {
			id, _ := uuid.NewV4()
			malformedOrigin := ""
			testEvent := &events.HttpStart{RequestId: factories.NewUUID(id)}
			envelope, err := emitter.Wrap(testEvent, malformedOrigin)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("Event not emitted due to missing origin information"))
			Expect(envelope).To(BeNil())
		})

		Context("with a known event type", func() {
			var testEvent events.Event

			BeforeEach(func() {
				id, _ := uuid.NewV4()
				testEvent = &events.HttpStop{RequestId: factories.NewUUID(id)}
			})

			It("should contain the origin", func() {
				envelope, _ := emitter.Wrap(testEvent, origin)
				Expect(envelope.GetOrigin()).To(Equal("testEventFormatter/42"))
			})

			Context("when the origin is empty", func() {
				It("should error with a helpful message", func() {
					envelope, err := emitter.Wrap(testEvent, "")
					Expect(envelope).To(BeNil())
					Expect(err.Error()).To(Equal("Event not emitted due to missing origin information"))
				})
			})
		})
	})
})
