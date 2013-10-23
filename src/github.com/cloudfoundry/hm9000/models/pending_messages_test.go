package models_test

import (
	. "github.com/cloudfoundry/hm9000/models"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"time"
)

var _ = Describe("Pending Messages", func() {
	Describe("Start Message", func() {
		var message PendingStartMessage
		BeforeEach(func() {
			message = NewPendingStartMessage(time.Unix(100, 0), 30, 10, "app-guid", "app-version", 1, 0.3)
		})

		It("should generate a random message id guid", func() {
			Ω(message.MessageId).ShouldNot(BeZero())
		})

		Describe("Creating new start messages programatically", func() {
			It("should populate the start message correctly, and compute the correct SendOn time", func() {
				Ω(message.SendOn).Should(BeNumerically("==", 130))
				Ω(message.SentOn).Should(BeNumerically("==", 0))
				Ω(message.KeepAlive).Should(BeNumerically("==", 10))
				Ω(message.AppGuid).Should(Equal("app-guid"))
				Ω(message.AppVersion).Should(Equal("app-version"))
				Ω(message.IndexToStart).Should(Equal(1))
				Ω(message.Priority).Should(Equal(0.3))
			})
		})

		Describe("Creating new start messages from JSON", func() {
			Context("when passed valid JSON", func() {
				It("should parse correctly", func() {
					parsed, err := NewPendingStartMessageFromJSON([]byte(`{
                        "send_on": 130,
                        "sent_on": 0,
                        "keep_alive": 10,
                        "droplet": "app-guid",
                        "version": "app-version",
                        "index": 1,
                        "message_id": "abc",
                        "priority": 0.3
                    }`))
					Ω(err).ShouldNot(HaveOccured())
					message.MessageId = "abc"
					Ω(parsed).Should(Equal(message))
				})
			})

			Context("when passed unparseable JSON", func() {
				It("should error", func() {
					parsed, err := NewPendingStartMessageFromJSON([]byte(`ß`))
					Ω(parsed).Should(BeZero())
					Ω(err).Should(HaveOccured())
				})
			})
		})

		Describe("ToJSON", func() {
			It("should generate valid JSON", func() {
				roundTripMessage, err := NewPendingStartMessageFromJSON(message.ToJSON())
				Ω(err).ShouldNot(HaveOccured())
				Ω(roundTripMessage).Should(Equal(message))
			})
		})

		Describe("StoreKey", func() {
			It("should generate the correct key", func() {
				Ω(message.StoreKey()).Should(Equal("app-guid-app-version-1"))
			})
		})

		Describe("LogDescription", func() {
			It("should generate an appropriate map", func() {
				Ω(message.LogDescription()).Should(Equal(map[string]string{
					"SendOn":       time.Unix(130, 0).String(),
					"SentOn":       time.Unix(0, 0).String(),
					"KeepAlive":    "10",
					"AppGuid":      "app-guid",
					"AppVersion":   "app-version",
					"IndexToStart": "1",
					"MessageId":    message.MessageId,
				}))
			})
		})

		Describe("Equality", func() {
			It("should work, and ignore the random MessageId", func() {
				anotherMessage := NewPendingStartMessage(time.Unix(100, 0), 30, 10, "app-guid", "app-version", 1, 0.3)
				Ω(message.Equal(anotherMessage)).Should(BeTrue())

				mutatedMessage := anotherMessage
				mutatedMessage.SendOn = 1
				Ω(message.Equal(mutatedMessage)).Should(BeFalse())

				mutatedMessage = anotherMessage
				mutatedMessage.SentOn = 1
				Ω(message.Equal(mutatedMessage)).Should(BeFalse())

				mutatedMessage = anotherMessage
				mutatedMessage.KeepAlive = 1
				Ω(message.Equal(mutatedMessage)).Should(BeFalse())

				mutatedMessage = anotherMessage
				mutatedMessage.AppGuid = "fluff"
				Ω(message.Equal(mutatedMessage)).Should(BeFalse())

				mutatedMessage = anotherMessage
				mutatedMessage.AppVersion = "bunny"
				Ω(message.Equal(mutatedMessage)).Should(BeFalse())

				mutatedMessage = anotherMessage
				mutatedMessage.IndexToStart = 17
				Ω(message.Equal(mutatedMessage)).Should(BeFalse())

				mutatedMessage = anotherMessage
				mutatedMessage.Priority = 3.141
				Ω(message.Equal(mutatedMessage)).Should(BeFalse())
			})
		})
	})

	Describe("Stop Message", func() {
		var message PendingStopMessage
		BeforeEach(func() {
			message = NewPendingStopMessage(time.Unix(100, 0), 30, 10, "app-guid", "app-version", "instance-guid")
		})

		It("should generate a random message id guid", func() {
			Ω(message.MessageId).ShouldNot(BeZero())
		})

		Describe("Creating new start messages programatically", func() {
			It("should populate the start message correctly, and compute the correct SendOn time", func() {
				Ω(message.SendOn).Should(BeNumerically("==", 130))
				Ω(message.SentOn).Should(BeNumerically("==", 0))
				Ω(message.KeepAlive).Should(BeNumerically("==", 10))
				Ω(message.AppGuid).Should(Equal("app-guid"))
				Ω(message.AppVersion).Should(Equal("app-version"))
				Ω(message.InstanceGuid).Should(Equal("instance-guid"))
			})
		})

		Describe("Creating new start messages from JSON", func() {
			Context("when passed valid JSON", func() {
				It("should parse correctly", func() {
					parsed, err := NewPendingStopMessageFromJSON([]byte(`{
                        "send_on": 130,
                        "sent_on": 0,
                        "keep_alive": 10,
                        "instance": "instance-guid",
                        "droplet": "app-guid",
                        "version": "app-version",
                        "message_id": "abc"
                    }`))
					Ω(err).ShouldNot(HaveOccured())
					message.MessageId = "abc"
					Ω(parsed).Should(Equal(message))
				})
			})

			Context("when passed unparseable JSON", func() {
				It("should error", func() {
					parsed, err := NewPendingStopMessageFromJSON([]byte(`ß`))
					Ω(parsed).Should(BeZero())
					Ω(err).Should(HaveOccured())
				})
			})
		})

		Describe("ToJSON", func() {
			It("should generate valid JSON", func() {
				roundTripMessage, err := NewPendingStopMessageFromJSON(message.ToJSON())
				Ω(err).ShouldNot(HaveOccured())
				Ω(roundTripMessage).Should(Equal(message))
			})
		})

		Describe("StoreKey", func() {
			It("should generate the correct key", func() {
				Ω(message.StoreKey()).Should(Equal("instance-guid"))
			})
		})

		Describe("LogDescription", func() {
			It("should generate an appropriate map", func() {
				Ω(message.LogDescription()).Should(Equal(map[string]string{
					"SendOn":       time.Unix(130, 0).String(),
					"SentOn":       time.Unix(0, 0).String(),
					"KeepAlive":    "10",
					"InstanceGuid": "instance-guid",
					"AppGuid":      "app-guid",
					"AppVersion":   "app-version",
					"MessageId":    message.MessageId,
				}))
			})
		})

		Describe("Equality", func() {
			It("should work, and ignore the random MessageId", func() {
				anotherMessage := NewPendingStopMessage(time.Unix(100, 0), 30, 10, "app-guid", "app-version", "instance-guid")
				Ω(message.Equal(anotherMessage)).Should(BeTrue())

				mutatedMessage := anotherMessage
				mutatedMessage.SendOn = 1
				Ω(message.Equal(mutatedMessage)).Should(BeFalse())

				mutatedMessage = anotherMessage
				mutatedMessage.SentOn = 1
				Ω(message.Equal(mutatedMessage)).Should(BeFalse())

				mutatedMessage = anotherMessage
				mutatedMessage.KeepAlive = 1
				Ω(message.Equal(mutatedMessage)).Should(BeFalse())

				mutatedMessage = anotherMessage
				mutatedMessage.InstanceGuid = "cheesecake"
				Ω(message.Equal(mutatedMessage)).Should(BeFalse())

				mutatedMessage = anotherMessage
				mutatedMessage.AppGuid = "pumpkin"
				Ω(message.Equal(mutatedMessage)).Should(BeFalse())

				mutatedMessage = anotherMessage
				mutatedMessage.AppVersion = "methuselah"
				Ω(message.Equal(mutatedMessage)).Should(BeFalse())
			})
		})
	})

	Describe("Pending Message", func() {
		var message PendingMessage
		BeforeEach(func() {
			message = PendingMessage{}
		})

		Context("when it was sent", func() {
			BeforeEach(func() {
				message.SentOn = 130
			})

			It("should be sent", func() {
				Ω(message.HasBeenSent()).Should(BeTrue())
			})
			Context("when keep alive time passed", func() {
				BeforeEach(func() {
					message.KeepAlive = 10
				})
				It("should be expired", func() {
					Ω(message.IsExpired(time.Unix(140, 0))).Should(BeTrue())
				})
			})
			Context("when keep alive time has not passed", func() {
				BeforeEach(func() {
					message.KeepAlive = 10
				})
				It("should not be expired", func() {
					Ω(message.IsExpired(time.Unix(139, 0))).Should(BeFalse())
				})
			})

			It("should not be ready to send", func() {
				Ω(message.IsTimeToSend(time.Unix(131, 0))).Should(BeFalse())
			})
		})

		Context("when it was not yet sent", func() {
			It("should not be sent", func() {
				Ω(message.HasBeenSent()).Should(BeFalse())
			})
			It("should not be expired", func() {
				Ω(message.IsExpired(time.Unix(129, 0))).Should(BeFalse())
			})
			Context("when send on time has passed", func() {
				BeforeEach(func() {
					message.SendOn = 130
				})
				It("should be ready to send", func() {
					Ω(message.IsTimeToSend(time.Unix(130, 0))).Should(BeTrue())
				})
			})
			Context("when send on time has not passed", func() {
				BeforeEach(func() {
					message.SendOn = 131
				})
				It("should not be ready to send", func() {
					Ω(message.IsTimeToSend(time.Unix(130, 0))).Should(BeFalse())
				})
			})
		})
	})
})
