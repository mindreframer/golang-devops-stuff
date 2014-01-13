package measurements_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMeasurements(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Measurements Suite")
}
