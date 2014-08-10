package registry_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestRegistry(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Registry Suite")
}
