package message_reader_test

import (
	"bufio"
	"bytes"

	"code.google.com/p/gogoprotobuf/proto"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf-experimental/garden/message_reader"
	protocol "github.com/pivotal-cf-experimental/garden/protocol"
)

var _ = Describe("Reading request messages over the wire", func() {
	Context("when a request is received", func() {
		It("returns the request and no error", func() {
			payload := bufio.NewReader(
				protocol.Messages(&protocol.EchoRequest{
					Message: proto.String("some-message"),
				}),
			)

			request, err := message_reader.ReadRequest(payload)
			Expect(err).ToNot(HaveOccurred())
			Expect(request).To(Equal(
				&protocol.EchoRequest{
					Message: proto.String("some-message"),
				},
			))
		})
	})

	Context("when the connection is broken", func() {
		It("returns an error", func() {
			payload := protocol.Messages(&protocol.PingRequest{})

			bogusPayload := bufio.NewReader(
				bytes.NewBuffer(payload.Bytes()[0 : payload.Len()-1]),
			)

			_, err := message_reader.ReadRequest(bogusPayload)
			Expect(err).To(HaveOccurred())
		})
	})
})
