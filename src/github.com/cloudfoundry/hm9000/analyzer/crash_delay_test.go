package analyzer_test

import (
	. "github.com/cloudfoundry/hm9000/analyzer"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CrashDelay", func() {
	var numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay int

	BeforeEach(func() {
		numberOfCrashesBeforeBackoffBegins = 3
		startingDelay = 30
		maximumDelay = 960
	})

	It("should return no delay until the crash count exceeds a threshold", func() {
		Ω(ComputeCrashDelay(0, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 0))
		Ω(ComputeCrashDelay(1, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 0))
		Ω(ComputeCrashDelay(2, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 0))
		Ω(ComputeCrashDelay(3, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 30))
	})

	It("should back-off the delay after the crash count exceeds the threshold", func() {
		Ω(ComputeCrashDelay(3, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 30))
		Ω(ComputeCrashDelay(4, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 60))
		Ω(ComputeCrashDelay(5, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 120))
		Ω(ComputeCrashDelay(6, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 240))
		Ω(ComputeCrashDelay(7, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 480))
		Ω(ComputeCrashDelay(8, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 960))
	})

	It("should plateau at the maximum for higher crash counts", func() {
		Ω(ComputeCrashDelay(8, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 960))
		Ω(ComputeCrashDelay(9, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 960))
		Ω(ComputeCrashDelay(10, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 960))
		Ω(ComputeCrashDelay(10000, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 960))
	})

	Context("when the maximum delay is not a power-of-two of the starting delay", func() {
		BeforeEach(func() {
			maximumDelay = 950
		})

		It("should still work appropriately", func() {
			Ω(ComputeCrashDelay(0, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 0))
			Ω(ComputeCrashDelay(1, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 0))
			Ω(ComputeCrashDelay(2, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 0))
			Ω(ComputeCrashDelay(3, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 30))
			Ω(ComputeCrashDelay(4, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 60))
			Ω(ComputeCrashDelay(5, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 120))
			Ω(ComputeCrashDelay(6, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 240))
			Ω(ComputeCrashDelay(7, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 480))
			Ω(ComputeCrashDelay(8, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 950))
			Ω(ComputeCrashDelay(9, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 950))
			Ω(ComputeCrashDelay(1000, numberOfCrashesBeforeBackoffBegins, startingDelay, maximumDelay)).Should(BeNumerically("==", 950))
		})
	})
})
