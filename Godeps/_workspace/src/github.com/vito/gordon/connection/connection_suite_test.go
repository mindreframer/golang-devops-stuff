package connection_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestConnection(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Connection Suite")
}
