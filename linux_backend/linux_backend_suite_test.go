package linux_backend_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestLinuxbackend(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Linux Backend Suite")
}
