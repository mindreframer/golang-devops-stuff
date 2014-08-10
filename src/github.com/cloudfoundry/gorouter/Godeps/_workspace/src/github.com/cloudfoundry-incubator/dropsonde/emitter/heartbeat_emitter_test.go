package emitter_test

import (
	"bytes"
	"code.google.com/p/gogoprotobuf/proto"
	"errors"
	"github.com/cloudfoundry-incubator/dropsonde/emitter"
	"github.com/cloudfoundry-incubator/dropsonde/emitter/fake"
	"github.com/cloudfoundry-incubator/dropsonde/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"log"
	"time"
)

var _ = Describe("HeartbeatEmitter", func() {
	var (
		wrappedEmitter *fake.FakeByteEmitter
		origin         = "testHeartbeatEmitter/0"
	)

	BeforeEach(func() {
		emitter.HeartbeatInterval = 10 * time.Millisecond
		wrappedEmitter = fake.NewFakeByteEmitter()
	})

	Describe("NewHeartbeatEmitter", func() {
		It("requires non-nil args", func() {
			hbEmitter, err := emitter.NewHeartbeatEmitter(nil, origin)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("wrappedEmitter is nil"))
			Expect(hbEmitter).To(BeNil())
		})

		It("starts periodic heartbeat emission", func() {
			hbEmitter, err := emitter.NewHeartbeatEmitter(wrappedEmitter, origin)
			Expect(err).NotTo(HaveOccurred())
			Expect(hbEmitter).NotTo(BeNil())

			Eventually(func() int { return len(wrappedEmitter.GetMessages()) }).Should(BeNumerically(">=", 2))
		})

		It("logs an error when heartbeat emission fails", func() {
			wrappedEmitter.ReturnError = errors.New("fake error")

			logWriter := new(bytes.Buffer)
			log.SetOutput(logWriter)

			hbEmitter, _ := emitter.NewHeartbeatEmitter(wrappedEmitter, origin)

			Eventually(func() int { return len(wrappedEmitter.GetMessages()) }).Should(BeNumerically(">=", 2))

			loggedText := string(logWriter.Bytes())
			expectedText := "fake error"
			Expect(loggedText).To(ContainSubstring(expectedText))
			hbEmitter.Close()
		})
	})

	Describe("Emit", func() {
		var (
			hbEmitter emitter.ByteEmitter
			testData  = []byte("hello")
		)

		BeforeEach(func() {
			hbEmitter, _ = emitter.NewHeartbeatEmitter(wrappedEmitter, origin)
		})

		It("delegates to the wrapped emitter", func() {
			hbEmitter.Emit(testData)

			messages := wrappedEmitter.GetMessages()
			Expect(messages).To(HaveLen(1))
			Expect(messages[0]).To(Equal(testData))
		})

		It("increments the heartbeat counter", func() {
			hbEmitter.Emit(testData)

			Eventually(func() bool {
				messages := wrappedEmitter.GetMessages()

				for _, message := range messages {
					hbEnvelope := &events.Envelope{}
					err := proto.Unmarshal(message, hbEnvelope)
					if err != nil || hbEnvelope.GetEventType() != events.Envelope_Heartbeat {
						continue // Not an envelope; keep looking
					}

					hbEvent := hbEnvelope.GetHeartbeat()

					if hbEvent.GetReceivedCount() == 1 {
						return true
					}
				}

				return false
			}).Should(BeTrue())
		})
	})

	Describe("Close", func() {
		var hbEmitter emitter.ByteEmitter

		BeforeEach(func() {
			hbEmitter, _ = emitter.NewHeartbeatEmitter(wrappedEmitter, origin)
		})

		It("eventually delegates to the inner heartbeat emitter", func() {
			hbEmitter.Close()
			Eventually(wrappedEmitter.IsClosed).Should(BeTrue())
		})

		It("can be called more than once", func() {
			hbEmitter.Close()
			Expect(hbEmitter.Close).ToNot(Panic())
		})
	})
})
