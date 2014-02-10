package gordon_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/vito/gordon"
	"net"
)

var _ = Describe("ConnectionProvider", func() {
	var listener net.Listener

	BeforeEach(func() {
		var err error

		listener, err = net.Listen("tcp", ":0")
		Ω(err).ShouldNot(HaveOccurred())
	})

	It("should connect to the listener", func() {
		info := &ConnectionInfo{
			Network: listener.Addr().Network(),
			Addr:    listener.Addr().String(),
		}

		conn, err := info.ProvideConnection()
		Ω(conn).ShouldNot(BeNil())
		Ω(err).ShouldNot(HaveOccurred())
	})
})
