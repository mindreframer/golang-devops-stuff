package container_pool_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestContainerPool(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Container Pool Suite")
}
