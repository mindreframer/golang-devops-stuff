package cmdtest_matchers_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/vito/cmdtest/matchers"
)

var _ = Describe("Say Matcher", func() {
	It("matches if the program outputs the expected string", func() {
		Expect(Run("echo", "-n", "hello")).To(Say("hello"))
	})

	It("matches if the program outputs a substring of the expected string", func() {
		Expect(Run("echo", "hello there")).To(Say("o t"))
	})

	It("does not match if the program does not output the expected string", func() {
		Expect(Run("echo", "hello")).NotTo(Say("goodbye"))
	})
})
