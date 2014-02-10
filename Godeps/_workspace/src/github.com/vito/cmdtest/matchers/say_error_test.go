package cmdtest_matchers_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/vito/cmdtest/matchers"
)

var _ = Describe("SayError Matcher", func() {
	It("matches if the program outputs the expected string", func() {
		Expect(Run("bash", "-c", "echo -n hello > &2")).To(SayError("hello"))
	})

	It("matches if the program outputs a substring of the expected string", func() {
		Expect(Run("bash", "-c", "echo hello there > &2")).To(SayError("o t"))
	})

	It("does not match if the program does not output the expected string", func() {
		Expect(Run("bash", "-c", "echo hello > &2")).NotTo(SayError("goodbye"))
	})

	It("does not match if the program outputs the expected string to standard out", func() {
		Expect(Run("echo", "hello")).NotTo(SayError("hello"))
	})
})
