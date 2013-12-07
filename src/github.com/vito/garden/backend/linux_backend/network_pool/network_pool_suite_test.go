package network_pool_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestNetwork_pool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Network Pool Suite")
}
