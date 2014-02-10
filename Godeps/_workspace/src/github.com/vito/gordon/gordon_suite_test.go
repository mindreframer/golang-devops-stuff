package gordon_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestGordon(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Gordon Suite")
}
