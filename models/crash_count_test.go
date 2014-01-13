package models_test

import (
	. "github.com/cloudfoundry/hm9000/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("CrashCount", func() {
	var crashCount CrashCount

	BeforeEach(func() {
		crashCount = CrashCount{
			AppGuid:       "abc",
			AppVersion:    "123",
			InstanceIndex: 1,
			CrashCount:    12,
			CreatedAt:     172,
		}
	})

	Describe("ToJSON", func() {
		It("should have the right fields", func() {
			json := string(crashCount.ToJSON())
			Ω(json).Should(ContainSubstring(`"droplet":"abc"`))
			Ω(json).Should(ContainSubstring(`"version":"123"`))
			Ω(json).Should(ContainSubstring(`"instance_index":1`))
			Ω(json).Should(ContainSubstring(`"crash_count":12`))
			Ω(json).Should(ContainSubstring(`"created_at":172`))
		})
	})

	Describe("NewCrashCountFromJSON", func() {
		It("should create right crash count", func() {
			decoded, err := NewCrashCountFromJSON(crashCount.ToJSON())
			Ω(err).ShouldNot(HaveOccurred())
			Ω(decoded).Should(Equal(crashCount))
		})

		It("should error when passed invalid json", func() {
			message, err := NewCrashCountFromJSON([]byte("∂"))
			Ω(message).Should(BeZero())
			Ω(err).Should(HaveOccurred())
		})
	})

	Describe("StoreKey", func() {
		It("should return appguid-appversion-index", func() {
			Ω(crashCount.StoreKey()).Should(Equal("abc-123-1"))
		})
	})
})
