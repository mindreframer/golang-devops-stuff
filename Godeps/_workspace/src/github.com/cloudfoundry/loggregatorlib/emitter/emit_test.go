package emitter_test

import (
	"code.google.com/p/gogoprotobuf/proto"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"
	. "github.com/cloudfoundry/loggregatorlib/emitter"
	"github.com/cloudfoundry/loggregatorlib/logmessage"
	"github.com/cloudfoundry/loggregatorlib/logmessage/testhelpers"
	"strings"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Testing with Ginkgo", func() {
	var (
		received chan *[]byte
		emitter  *LoggregatorEmitter
	)

	BeforeEach(func() {
		var err error
		received = make(chan *[]byte, 10)
		emitter, err = NewEmitter("localhost:3456", "ROUTER", "42", "secret", nil)
		Ω(err).ShouldNot(HaveOccurred())

		emitter.LoggregatorClient = &MockLoggregatorClient{received}

	})

	It("should emit stdout", func() {
		emitter.Emit("appid", "foo")
		receivedMessage := extractLogMessage(<-received)

		Ω(receivedMessage.GetMessage()).Should(Equal([]byte("foo")))
		Ω(receivedMessage.GetAppId()).Should(Equal("appid"))
		Ω(receivedMessage.GetSourceId()).Should(Equal("42"))
		Ω(receivedMessage.GetMessageType()).Should(Equal(logmessage.LogMessage_OUT))
	})

	It("should emit stderr", func() {
		emitter.EmitError("appid", "foo")
		receivedMessage := extractLogMessage(<-received)

		Ω(receivedMessage.GetMessage()).Should(Equal([]byte("foo")))
		Ω(receivedMessage.GetAppId()).Should(Equal("appid"))
		Ω(receivedMessage.GetSourceId()).Should(Equal("42"))
		Ω(receivedMessage.GetMessageType()).Should(Equal(logmessage.LogMessage_ERR))
	})

	It("should emit fully formed log messages", func() {
		logMessage := testhelpers.NewLogMessage("test_msg", "test_app_id")
		logMessage.SourceId = proto.String("src_id")

		emitter.EmitLogMessage(logMessage)
		receivedMessage := extractLogMessage(<-received)

		Ω(receivedMessage.GetMessage()).Should(Equal([]byte("test_msg")))
		Ω(receivedMessage.GetAppId()).Should(Equal("test_app_id"))
		Ω(receivedMessage.GetSourceId()).Should(Equal("src_id"))
	})

	It("should truncate long messages", func() {
		longMessage := strings.Repeat("7", MAX_MESSAGE_BYTE_SIZE*2)
		logMessage := testhelpers.NewLogMessage(longMessage, "test_app_id")

		emitter.EmitLogMessage(logMessage)

		receivedMessage := extractLogMessage(<-received)
		receivedMessageText := receivedMessage.GetMessage()

		truncatedOffset := len(receivedMessageText) - len(TRUNCATED_BYTES)
		expectedBytes := append([]byte(receivedMessageText)[:truncatedOffset], TRUNCATED_BYTES...)

		Ω(receivedMessageText).Should(Equal(expectedBytes))
		Ω(receivedMessageText).Should(HaveLen(MAX_MESSAGE_BYTE_SIZE))
	})

	It("should split messages on new lines", func() {
		message := "message1\n\rmessage2\nmessage3\r\nmessage4\r"
		logMessage := testhelpers.NewLogMessage(message, "test_app_id")

		emitter.EmitLogMessage(logMessage)
		Ω(received).Should(HaveLen(4))

		for _, expectedMessage := range []string{"message1", "message2", "message3", "message4"} {
			receivedMessage := extractLogMessage(<-received)
			Ω(receivedMessage.GetMessage()).Should(Equal([]byte(expectedMessage)))
		}
	})

	It("should build the log envelope correctly", func() {
		emitter.Emit("appid", "foo")
		receivedEnvelope := extractLogEnvelope(<-received)

		Ω(receivedEnvelope.GetLogMessage().GetMessage()).Should(Equal([]byte("foo")))
		Ω(receivedEnvelope.GetLogMessage().GetAppId()).Should(Equal("appid"))
		Ω(receivedEnvelope.GetRoutingKey()).Should(Equal("appid"))
		Ω(receivedEnvelope.GetLogMessage().GetSourceId()).Should(Equal("42"))
	})

	It("should sign the log message correctly", func() {
		emitter.Emit("appid", "foo")
		receivedEnvelope := extractLogEnvelope(<-received)
		Ω(receivedEnvelope.VerifySignature("secret")).Should(BeTrue(), "Expected envelope to be signed with the correct secret key")
	})

	It("source name is set if mapping is unknown", func() {
		emitter, err := NewEmitter("localhost:3456", "XYZ", "42", "secret", nil)
		Ω(err).ShouldNot(HaveOccurred())
		emitter.LoggregatorClient = &MockLoggregatorClient{received}

		emitter.Emit("test_app_id", "test_msg")
		receivedMessage := extractLogMessage(<-received)

		Ω(receivedMessage.GetSourceName()).Should(Equal("XYZ"))
	})

	Context("when missing an app id", func() {
		It("should not emit", func() {
			emitter.Emit("", "foo")
			Ω(received).ShouldNot(Receive(), "Message without app id should not have been emitted")

			emitter.Emit("    ", "foo")
			Ω(received).ShouldNot(Receive(), "Message with an empty app id should not have been emitted")
		})
	})
})

type MockLoggregatorClient struct {
	received chan *[]byte
}

func (m MockLoggregatorClient) Send(data []byte) {
	m.received <- &data
}

func (m MockLoggregatorClient) Emit() instrumentation.Context {
	return instrumentation.Context{}
}

func extractLogEnvelope(data *[]byte) *logmessage.LogEnvelope {
	receivedEnvelope := &logmessage.LogEnvelope{}

	err := proto.Unmarshal(*data, receivedEnvelope)
	Ω(err).ShouldNot(HaveOccurred())

	return receivedEnvelope
}

func extractLogMessage(data *[]byte) *logmessage.LogMessage {
	envelope := extractLogEnvelope(data)

	return envelope.GetLogMessage()
}
