package shredder_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestShredder(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Shredder Suite")
}
