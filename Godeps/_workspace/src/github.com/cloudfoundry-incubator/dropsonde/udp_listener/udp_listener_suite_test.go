package udp_listener_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestUdpListener(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "UdpListener Suite")
}
