package metricsaccountant_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMetricsAccountant(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Metricsaccountant Suite")
}
