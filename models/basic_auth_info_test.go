package models_test

import (
	"encoding/base64"
	. "github.com/cloudfoundry/hm9000/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("BasicAuthInfo", func() {
	Describe("basic auth encoding", func() {
		It("should encode the user and password in basic auth form", func() {
			info := BasicAuthInfo{"mcat", "testing"}
			Ω(info.Encode()).Should(Equal("Basic bWNhdDp0ZXN0aW5n"))
		})
	})

	Describe("basic auth decoding", func() {
		Context("when the string is malformed", func() {
			It("should return an error", func() {
				authInfo, err := DecodeBasicAuthInfo("bWNhdDp0ZXN0aW5n")
				Ω(authInfo).Should(BeZero())
				Ω(err).Should(HaveOccurred())

				authInfo, err = DecodeBasicAuthInfo("Basic " + base64.StdEncoding.EncodeToString([]byte("pink-flamingoes")))
				Ω(authInfo).Should(BeZero())
				Ω(err).Should(HaveOccurred())

				authInfo, err = DecodeBasicAuthInfo("Basic " + base64.StdEncoding.EncodeToString([]byte("pink-flamingoes:password:oops")))
				Ω(authInfo).Should(BeZero())
				Ω(err).Should(HaveOccurred())

				authInfo, err = DecodeBasicAuthInfo("Basic " + base64.StdEncoding.EncodeToString([]byte("pink-flamingoes:password")) + " oops")
				Ω(authInfo).Should(BeZero())
				Ω(err).Should(HaveOccurred())
			})
		})

		Context("when all is well", func() {
			It("should decode the user and password and not return an error", func() {
				authInfo, err := DecodeBasicAuthInfo("Basic bWNhdDp0ZXN0aW5n")
				Ω(err).ShouldNot(HaveOccurred())

				Ω(authInfo.User).Should(Equal("mcat"))
				Ω(authInfo.Password).Should(Equal("testing"))
			})
		})
	})
})
