package fakestoreadapter_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestFakestoreadapter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Fakestoreadapter Suite")
}
