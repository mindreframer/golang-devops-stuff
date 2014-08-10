package common_test

import (
	. "github.com/cloudfoundry/gorouter/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("common", func() {
	It("createsa uuid", func() {
		uuid, err := GenerateUUID()
		Ω(err).ShouldNot(HaveOccurred())
		Ω(uuid).Should(HaveLen(36))
	})
})
