package bandwidth_manager_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestBandwidth_manager(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Bandwidth_manager Suite")
}
