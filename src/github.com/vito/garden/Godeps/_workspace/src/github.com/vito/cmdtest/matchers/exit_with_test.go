package cmdtest_matchers_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/vito/cmdtest/matchers"
)

var _ = Describe("ExitWith Matcher", func() {
	It("matches if the program exits with the expected status", func() {
		// I hope this never happens.
		Expect(Run("ls", "/that/is/one/unlikely/path")).To(ExitWith(1))
	})

	It("does not match if the program does not exit with the expected status", func() {
		Expect(Run("echo", "SUCCESS")).NotTo(ExitWith(2))
	})
})
