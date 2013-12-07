package message_reader_test

import (
	"bufio"
	"bytes"

	"code.google.com/p/gogoprotobuf/proto"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vito/garden/message_reader"
	protocol "github.com/vito/garden/protocol"
)

var _ = Describe("Reading response messages over the wire", func() {
	Context("when a message of the expected type is received", func() {
		It("populates the response object and returns no error", func() {
			var echoResponse protocol.EchoResponse

			err := message_reader.ReadMessage(
				bufio.NewReader(protocol.Messages(&protocol.EchoRequest{
					Message: proto.String("some message"),
				})),
				&echoResponse,
			)

			Expect(err).ToNot(HaveOccured())

			Expect(echoResponse.GetMessage()).To(Equal("some message"))
		})
	})

	Context("when the connection is broken", func() {
		It("returns an error", func() {
			var dummyResponse protocol.PingResponse

			payload := protocol.Messages(&protocol.PingRequest{})

			bogusPayload := bufio.NewReader(
				bytes.NewBuffer(payload.Bytes()[0 : payload.Len()-1]),
			)

			err := message_reader.ReadMessage(bogusPayload, &dummyResponse)
			Expect(err).To(HaveOccured())
		})
	})

	Context("when an error is received", func() {
		It("returns a WardenError", func() {
			var dummyResponse protocol.PingResponse

			err := message_reader.ReadMessage(
				bufio.NewReader(protocol.Messages(&protocol.ErrorResponse{
					Message: proto.String("some message"),
					Data:    proto.String("some data"),
					Backtrace: []string{
						"backtrace line 1",
						"backtrace line 2",
					},
				})),
				&dummyResponse,
			)

			Expect(err).To(Equal(
				&message_reader.WardenError{
					Message: "some message",
					Data:    "some data",
					Backtrace: []string{
						"backtrace line 1",
						"backtrace line 2",
					},
				},
			))
		})
	})

	Context("when a message of the wrong type is received", func() {
		It("returns a TypeMismatchError", func() {
			var dummyResponse protocol.PingResponse

			actualResponse := &protocol.EchoResponse{
				Message: proto.String("some message"),
			}

			err := message_reader.ReadMessage(
				bufio.NewReader(protocol.Messages(actualResponse)),
				&dummyResponse,
			)

			Expect(err).To(Equal(
				&message_reader.TypeMismatchError{
					Expected: protocol.TypeForMessage(&dummyResponse),
					Received: protocol.TypeForMessage(actualResponse),
				},
			))
		})
	})
})
