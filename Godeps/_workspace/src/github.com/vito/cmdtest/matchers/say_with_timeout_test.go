package cmdtest_matchers_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/vito/cmdtest/matchers"
)

var _ = Describe("SayWithTimeout Matcher", func() {
	It("matches if the program outputs the expected string within the timeout", func() {
		Expect(Run("bash", "-c", "sleep 1 && echo hello")).To(SayWithTimeout("hello", 2*time.Second))
	})

	It("does not match if the program outputs does not output the expected string within the timeout", func() {
		Expect(Run("bash", "-c", "sleep 1 && echo hello")).NotTo(SayWithTimeout("hello", 500*time.Millisecond))
	})
})
