package cmdtest_matchers_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vito/cmdtest"
	. "github.com/vito/cmdtest/matchers"
)

var _ = Describe("SayBranches Matcher", func() {
	It("matches and runs the only code associated with the expected string", func() {
		var helloCalled bool = false
		var goodbyeCalled bool = false

		Expect(Run("echo", "hello")).To(SayBranches(
			cmdtest.ExpectBranch{
				Pattern: "hello",
				Callback: func() {
					helloCalled = true
				},
			},
			cmdtest.ExpectBranch{
				Pattern: "goodbye",
				Callback: func() {
					goodbyeCalled = true
				},
			},
		))

		Expect(helloCalled).To(BeTrue())
		Expect(goodbyeCalled).To(BeFalse())
	})

	Context("if more than one branch matches", func() {
		It("matches and runs the first associated with the expected string", func() {
			var firstCalled bool = false
			var secondCalled bool = false

			Expect(Run("echo", "hello")).To(SayBranches(
				cmdtest.ExpectBranch{
					Pattern: "hel",
					Callback: func() {
						firstCalled = true
					},
				},
				cmdtest.ExpectBranch{
					Pattern: "hello",
					Callback: func() {
						secondCalled = true
					},
				},
			))

			Expect(firstCalled).To(BeTrue())
			Expect(secondCalled).To(BeFalse())
		})
	})
})
