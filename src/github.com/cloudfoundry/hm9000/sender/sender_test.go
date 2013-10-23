package sender_test

import (
	"errors"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/models"
	. "github.com/cloudfoundry/hm9000/sender"
	storepackage "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	"github.com/cloudfoundry/hm9000/testhelpers/fakestoreadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/faketimeprovider"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Sender", func() {
	var (
		storeAdapter *fakestoreadapter.FakeStoreAdapter
		store        storepackage.Store
		sender       *Sender
		messageBus   *fakeyagnats.FakeYagnats
		timeProvider *faketimeprovider.FakeTimeProvider
		app1         appfixture.AppFixture
		conf         config.Config
	)

	BeforeEach(func() {
		messageBus = fakeyagnats.New()
		app1 = appfixture.NewAppFixture()
		conf, _ = config.DefaultConfig()

		timeProvider = &faketimeprovider.FakeTimeProvider{
			TimeToProvide: time.Unix(int64(10+conf.ActualFreshnessTTL()), 0),
		}

		storeAdapter = fakestoreadapter.New()
		store = storepackage.NewStore(conf, storeAdapter, fakelogger.NewFakeLogger())
		sender = New(store, conf, messageBus, timeProvider, fakelogger.NewFakeLogger())
		store.BumpActualFreshness(time.Unix(10, 0))
		store.BumpDesiredFreshness(time.Unix(10, 0))
	})

	Context("when the sender fails to pull messages out of the start queue", func() {
		BeforeEach(func() {
			storeAdapter.ListErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector("start", errors.New("oops"))
		})

		It("should return an error and not send any messages", func() {
			err := sender.Send()
			Ω(err).Should(Equal(errors.New("oops")))
			Ω(messageBus.PublishedMessages).Should(BeEmpty())
		})
	})

	Context("when the sender fails to pull messages out of the stop queue", func() {
		BeforeEach(func() {
			storeAdapter.ListErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector("stop", errors.New("oops"))
		})

		It("should return an error and not send any messages", func() {
			err := sender.Send()
			Ω(err).Should(Equal(errors.New("oops")))
			Ω(messageBus.PublishedMessages).Should(BeEmpty())
		})
	})

	Context("when the sender fails to fetch from the store", func() {
		BeforeEach(func() {
			storeAdapter.ListErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector("apps", errors.New("oops"))
		})

		It("should return an error and not send any messages", func() {
			err := sender.Send()
			Ω(err).Should(Equal(errors.New("oops")))
			Ω(messageBus.PublishedMessages).Should(BeEmpty())
		})
	})

	Context("when there are no start messages in the queue", func() {
		It("should not send any messages", func() {
			err := sender.Send()
			Ω(err).ShouldNot(HaveOccured())
			Ω(messageBus.PublishedMessages).Should(BeEmpty())
		})
	})

	Context("when there are no stop messages in the queue", func() {
		It("should not send any messages", func() {
			err := sender.Send()
			Ω(err).ShouldNot(HaveOccured())
			Ω(messageBus.PublishedMessages).Should(BeEmpty())
		})
	})

	Context("when there are start messages", func() {
		var keepAliveTime int
		var sentOn int64
		var err error
		var pendingMessage models.PendingStartMessage
		var storeSetErrInjector *fakestoreadapter.FakeStoreAdapterErrorInjector

		JustBeforeEach(func() {
			store.SaveDesiredState(app1.DesiredState(1))
			pendingMessage = models.NewPendingStartMessage(time.Unix(100, 0), 30, keepAliveTime, app1.AppGuid, app1.AppVersion, 0, 1.0)
			pendingMessage.SentOn = sentOn
			store.SavePendingStartMessages(
				pendingMessage,
			)
			storeAdapter.SetErrInjector = storeSetErrInjector
			err = sender.Send()
		})

		BeforeEach(func() {
			keepAliveTime = 0
			sentOn = 0
			err = nil
			storeSetErrInjector = nil
		})

		Context("and it is not time to send the message yet", func() {
			BeforeEach(func() {
				timeProvider.TimeToProvide = time.Unix(129, 0)
			})

			It("should not error", func() {
				Ω(err).ShouldNot(HaveOccured())
			})

			It("should not send the messages", func() {
				Ω(messageBus.PublishedMessages).ShouldNot(HaveKey("hm9000.start"))
			})

			It("should leave the messages in the queue", func() {
				messages, _ := store.GetPendingStartMessages()
				Ω(messages).Should(HaveLen(1))
			})
		})

		Context("and it is time to send the message", func() {
			BeforeEach(func() {
				timeProvider.TimeToProvide = time.Unix(130, 0)
			})

			It("should send the message", func() {
				Ω(messageBus.PublishedMessages["hm9000.start"]).Should(HaveLen(1))
				message, _ := models.NewStartMessageFromJSON([]byte(messageBus.PublishedMessages["hm9000.start"][0].Payload))
				Ω(message).Should(Equal(models.StartMessage{
					AppGuid:       app1.AppGuid,
					AppVersion:    app1.AppVersion,
					InstanceIndex: 0,
					MessageId:     pendingMessage.MessageId,
				}))
			})

			It("should not error", func() {
				Ω(err).ShouldNot(HaveOccured())
			})

			Context("when the message should be kept alive", func() {
				BeforeEach(func() {
					keepAliveTime = 30
				})

				It("should update the sent on times", func() {
					messages, _ := store.GetPendingStartMessages()
					Ω(messages).Should(HaveLen(1))
					for _, message := range messages {
						Ω(message.SentOn).Should(Equal(timeProvider.Time().Unix()))
					}
				})

				Context("when saving the start messages fails", func() {
					BeforeEach(func() {
						storeSetErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector("start", errors.New("oops"))
					})

					It("should return an error", func() {
						Ω(err).Should(Equal(errors.New("oops")))
					})
				})
			})

			Context("when the KeepAlive = 0", func() {
				BeforeEach(func() {
					keepAliveTime = 0
				})

				It("should just delete the message after sending it", func() {
					messages, _ := store.GetPendingStartMessages()
					Ω(messages).Should(BeEmpty())
				})

				Context("when deleting the start messages fails", func() {
					BeforeEach(func() {
						storeAdapter.DeleteErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector("start", errors.New("oops"))
					})

					It("should return an error", func() {
						Ω(err).Should(Equal(errors.New("oops")))
					})
				})
			})

			Context("when the message fails to send", func() {
				BeforeEach(func() {
					messageBus.PublishError = errors.New("oops")
				})

				It("should return an error", func() {
					Ω(err).Should(Equal(errors.New("oops")))
				})
			})
		})

		Context("When the message has already been sent", func() {
			BeforeEach(func() {
				sentOn = 130
				keepAliveTime = 30
			})

			Context("and the keep alive has elapsed", func() {
				BeforeEach(func() {
					timeProvider.TimeToProvide = time.Unix(160, 0)
				})

				It("should delete the message and not send it", func() {
					messages, _ := store.GetPendingStartMessages()
					Ω(messages).Should(BeEmpty())
					Ω(messageBus.PublishedMessages).ShouldNot(HaveKey("hm9000.start"))
				})
			})

			Context("and the keep alive has not elapsed", func() {
				BeforeEach(func() {
					timeProvider.TimeToProvide = time.Unix(159, 0)
				})

				It("should neither delete the message nor send it", func() {
					messages, _ := store.GetPendingStartMessages()
					Ω(messages).Should(HaveLen(1))

					Ω(messageBus.PublishedMessages).ShouldNot(HaveKey("hm9000.start"))
				})
			})
		})
	})

	Context("when there are stop messages", func() {
		var keepAliveTime int
		var sentOn int64
		var err error
		var pendingMessage models.PendingStopMessage
		var storeSetErrInjector *fakestoreadapter.FakeStoreAdapterErrorInjector

		JustBeforeEach(func() {
			store.SaveActualState(
				app1.InstanceAtIndex(0).Heartbeat(),
				app1.InstanceAtIndex(1).Heartbeat(),
			)

			pendingMessage = models.NewPendingStopMessage(time.Unix(100, 0), 30, keepAliveTime, app1.AppGuid, app1.AppVersion, app1.InstanceAtIndex(0).InstanceGuid)
			pendingMessage.SentOn = sentOn
			store.SavePendingStopMessages(
				pendingMessage,
			)

			storeAdapter.SetErrInjector = storeSetErrInjector
			err = sender.Send()
		})

		BeforeEach(func() {
			keepAliveTime = 0
			sentOn = 0
			err = nil
			storeSetErrInjector = nil
		})

		Context("and it is not time to send the message yet", func() {
			BeforeEach(func() {
				timeProvider.TimeToProvide = time.Unix(129, 0)
			})

			It("should not error", func() {
				Ω(err).ShouldNot(HaveOccured())
			})

			It("should not send the messages", func() {
				Ω(messageBus.PublishedMessages).ShouldNot(HaveKey("hm9000.stop"))
			})

			It("should leave the messages in the queue", func() {
				messages, _ := store.GetPendingStopMessages()
				Ω(messages).Should(HaveLen(1))
			})
		})

		Context("and it is time to send the message", func() {
			BeforeEach(func() {
				timeProvider.TimeToProvide = time.Unix(130, 0)
			})

			It("should not error", func() {
				Ω(err).ShouldNot(HaveOccured())
			})

			It("should send the message", func() {
				Ω(messageBus.PublishedMessages["hm9000.stop"]).Should(HaveLen(1))
				message, _ := models.NewStopMessageFromJSON([]byte(messageBus.PublishedMessages["hm9000.stop"][0].Payload))
				Ω(message).Should(Equal(models.StopMessage{
					AppGuid:       app1.AppGuid,
					AppVersion:    app1.AppVersion,
					InstanceIndex: 0,
					InstanceGuid:  app1.InstanceAtIndex(0).InstanceGuid,
					IsDuplicate:   false,
					MessageId:     pendingMessage.MessageId,
				}))
			})

			Context("when the message should be kept alive", func() {
				BeforeEach(func() {
					keepAliveTime = 30
				})

				It("should update the sent on times", func() {
					messages, _ := store.GetPendingStopMessages()
					Ω(messages).Should(HaveLen(1))
					for _, message := range messages {
						Ω(message.SentOn).Should(Equal(timeProvider.Time().Unix()))
					}
				})

				Context("when saving the stop message fails", func() {
					BeforeEach(func() {
						storeSetErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector("stop", errors.New("oops"))
					})

					It("should return an error", func() {
						Ω(err).Should(Equal(errors.New("oops")))
					})
				})
			})

			Context("when the KeepAlive = 0", func() {
				BeforeEach(func() {
					keepAliveTime = 0
				})

				It("should just delete the message after sending it", func() {
					messages, _ := store.GetPendingStopMessages()
					Ω(messages).Should(BeEmpty())
				})

				Context("when deleting the message fails", func() {
					BeforeEach(func() {
						storeAdapter.DeleteErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector("stop", errors.New("oops"))
					})

					It("should return an error", func() {
						Ω(err).Should(Equal(errors.New("oops")))
					})
				})
			})

			Context("when the message fails to send", func() {
				BeforeEach(func() {
					messageBus.PublishError = errors.New("oops")
				})

				It("should return an error", func() {
					Ω(err).Should(Equal(errors.New("oops")))
				})
			})
		})

		Context("When the message has already been sent", func() {
			BeforeEach(func() {
				sentOn = 130
				keepAliveTime = 30
			})

			It("should not error", func() {
				Ω(err).ShouldNot(HaveOccured())
			})

			Context("and the keep alive has elapsed", func() {
				BeforeEach(func() {
					timeProvider.TimeToProvide = time.Unix(160, 0)
				})

				It("should delete the message and not send it", func() {
					messages, _ := store.GetPendingStopMessages()
					Ω(messages).Should(BeEmpty())
					Ω(messageBus.PublishedMessages).ShouldNot(HaveKey("hm9000.stop"))
				})
			})

			Context("and the keep alive has not elapsed", func() {
				BeforeEach(func() {
					timeProvider.TimeToProvide = time.Unix(159, 0)
				})

				It("should neither delete the message nor send it", func() {
					messages, _ := store.GetPendingStopMessages()
					Ω(messages).Should(HaveLen(1))

					Ω(messageBus.PublishedMessages).ShouldNot(HaveKey("hm9000.stop"))
				})
			})
		})
	})

	Describe("Verifying that start messages should be sent", func() {
		var err error
		var indexToStart int
		var pendingMessage models.PendingStartMessage

		JustBeforeEach(func() {
			timeProvider.TimeToProvide = time.Unix(130, 0)
			pendingMessage = models.NewPendingStartMessage(time.Unix(100, 0), 30, 10, app1.AppGuid, app1.AppVersion, indexToStart, 1.0)
			pendingMessage.SentOn = 0
			store.SavePendingStartMessages(
				pendingMessage,
			)

			err = sender.Send()
		})

		BeforeEach(func() {
			err = nil
			indexToStart = 0
		})

		assertMessageWasNotSent := func() {
			It("should ignore the keep-alive and delete the start message from queue", func() {
				messages, _ := store.GetPendingStartMessages()
				Ω(messages).Should(HaveLen(0))
			})

			It("should not send the start message", func() {
				Ω(messageBus.PublishedMessages).ShouldNot(HaveKey("hm9000.start"))
			})
		}

		assertMessageWasSent := func() {
			It("should honor the keep alive of the start message", func() {
				messages, _ := store.GetPendingStartMessages()
				Ω(messages).Should(HaveLen(1))
				for _, message := range messages {
					Ω(message.SentOn).Should(BeNumerically("==", 130))
				}
			})

			It("should send the start message", func() {
				Ω(messageBus.PublishedMessages["hm9000.start"]).Should(HaveLen(1))
				message, _ := models.NewStartMessageFromJSON([]byte(messageBus.PublishedMessages["hm9000.start"][0].Payload))
				Ω(message).Should(Equal(models.StartMessage{
					AppGuid:       app1.AppGuid,
					AppVersion:    app1.AppVersion,
					InstanceIndex: 0,
					MessageId:     pendingMessage.MessageId,
				}))
			})
		}

		Context("When the app is still desired", func() {
			BeforeEach(func() {
				store.SaveDesiredState(app1.DesiredState(1))
			})

			Context("when the index-to-start is within the # of desired instances", func() {
				BeforeEach(func() {
					indexToStart = 0
				})

				Context("when there are no running instances at all for that app", func() {
					assertMessageWasSent()
				})

				Context("when there is no running instance reporting at that index", func() {
					BeforeEach(func() {
						store.SaveActualState(
							app1.InstanceAtIndex(1).Heartbeat(),
							app1.InstanceAtIndex(2).Heartbeat(),
						)
					})
					assertMessageWasSent()
				})

				Context("when there are crashed instances reporting at that index", func() {
					BeforeEach(func() {
						store.SaveActualState(
							app1.CrashedInstanceHeartbeatAtIndex(0),
							app1.CrashedInstanceHeartbeatAtIndex(0),
							app1.InstanceAtIndex(1).Heartbeat(),
							app1.InstanceAtIndex(2).Heartbeat(),
						)
					})

					assertMessageWasSent()
				})

				Context("when there *is* a running instance reporting at that index", func() {
					BeforeEach(func() {
						store.SaveActualState(
							app1.InstanceAtIndex(0).Heartbeat(),
						)
					})

					assertMessageWasNotSent()
				})
			})

			Context("when the index-to-start is beyond the # of desired instances", func() {
				BeforeEach(func() {
					indexToStart = 1
				})

				assertMessageWasNotSent()
			})
		})

		Context("When the app is no longer desired", func() {
			assertMessageWasNotSent()
		})
	})

	Describe("Verifying that stop messages should be sent", func() {
		var err error
		var indexToStop int
		var pendingMessage models.PendingStopMessage

		JustBeforeEach(func() {
			timeProvider.TimeToProvide = time.Unix(130, 0)
			pendingMessage = models.NewPendingStopMessage(time.Unix(100, 0), 30, 10, app1.AppGuid, app1.AppVersion, app1.InstanceAtIndex(indexToStop).InstanceGuid)
			pendingMessage.SentOn = 0
			store.SavePendingStopMessages(
				pendingMessage,
			)

			err = sender.Send()
		})

		BeforeEach(func() {
			indexToStop = 0
		})

		assertMessageWasNotSent := func() {
			It("should ignore the keep-alive and delete the stop message from queue", func() {
				messages, _ := store.GetPendingStopMessages()
				Ω(messages).Should(HaveLen(0))
			})

			It("should not send the stop message", func() {
				Ω(messageBus.PublishedMessages).ShouldNot(HaveKey("hm9000.stop"))
			})
		}

		assertMessageWasSent := func(indexToStop int, isDuplicate bool) {
			It("should honor the keep alive of the stop message", func() {
				messages, _ := store.GetPendingStopMessages()
				Ω(messages).Should(HaveLen(1))
				for _, message := range messages {
					Ω(message.SentOn).Should(BeNumerically("==", 130))
				}

			})

			It("should send the stop message", func() {
				Ω(messageBus.PublishedMessages["hm9000.stop"]).Should(HaveLen(1))
				message, _ := models.NewStopMessageFromJSON([]byte(messageBus.PublishedMessages["hm9000.stop"][0].Payload))
				Ω(message).Should(Equal(models.StopMessage{
					AppGuid:       app1.AppGuid,
					AppVersion:    app1.AppVersion,
					InstanceIndex: indexToStop,
					InstanceGuid:  app1.InstanceAtIndex(indexToStop).InstanceGuid,
					IsDuplicate:   isDuplicate,
					MessageId:     pendingMessage.MessageId,
				}))
			})
		}

		Context("When the app is still desired", func() {
			BeforeEach(func() {
				store.SaveDesiredState(app1.DesiredState(1))
			})

			Context("When index is still running", func() {
				BeforeEach(func() {
					store.SaveActualState(
						app1.InstanceAtIndex(0).Heartbeat(),
						app1.InstanceAtIndex(1).Heartbeat(),
					)
				})

				Context("When index-to-stop is within the number of desired instances", func() {
					BeforeEach(func() {
						indexToStop = 0
					})

					Context("When there are other running instances on the index", func() {
						BeforeEach(func() {
							instance := app1.InstanceAtIndex(0)
							instance.InstanceGuid = models.Guid()

							store.SaveActualState(
								instance.Heartbeat(),
							)
						})

						assertMessageWasSent(0, true)
					})

					Context("when there are other, crashed, instances on the index", func() {
						BeforeEach(func() {
							store.SaveActualState(
								app1.CrashedInstanceHeartbeatAtIndex(0),
							)
						})

						assertMessageWasNotSent()
					})

					Context("When there are no other running instances on the index", func() {
						assertMessageWasNotSent()
					})
				})

				Context("When index-to-stop is beyond the number of desired instances", func() {
					BeforeEach(func() {
						indexToStop = 1
					})

					assertMessageWasSent(1, false)
				})
			})

			Context("When index is not running", func() {
				assertMessageWasNotSent()
			})
		})

		Context("When the app is no longer desired", func() {
			Context("when the instance is still running", func() {
				BeforeEach(func() {
					store.SaveActualState(
						app1.InstanceAtIndex(0).Heartbeat(),
						app1.InstanceAtIndex(1).Heartbeat(),
					)
				})
				assertMessageWasSent(0, false)
			})

			Context("when the instance is not running", func() {
				assertMessageWasNotSent()
			})
		})
	})

	Context("When there are multiple start and stop messages in the queue", func() {
		var invalidStartMessages, validStartMessages, expiredStartMessages []models.PendingStartMessage

		BeforeEach(func() {
			conf, _ = config.DefaultConfig()
			conf.SenderMessageLimit = 20

			sender = New(store, conf, messageBus, timeProvider, fakelogger.NewFakeLogger())

			for i := 0; i < 40; i += 1 {
				a := appfixture.NewAppFixture()
				store.SaveDesiredState(a.DesiredState(1))
				store.SaveActualState(
					a.InstanceAtIndex(1).Heartbeat(),
				)

				//only some of these should be sent:
				validStartMessage := models.NewPendingStartMessage(time.Unix(100, 0), 30, 0, a.AppGuid, a.AppVersion, 0, float64(i)/40.0)
				validStartMessages = append(validStartMessages, validStartMessage)
				store.SavePendingStartMessages(
					validStartMessage,
				)

				//all of these should be deleted:
				invalidStartMessage := models.NewPendingStartMessage(time.Unix(100, 0), 30, 0, a.AppGuid, a.AppVersion, 1, 1.0)
				invalidStartMessages = append(invalidStartMessages, invalidStartMessage)
				store.SavePendingStartMessages(
					invalidStartMessage,
				)

				//all of these should be deleted:
				expiredStartMessage := models.NewPendingStartMessage(time.Unix(100, 0), 0, 20, a.AppGuid, a.AppVersion, 2, 1.0)
				expiredStartMessage.SentOn = 100
				expiredStartMessages = append(expiredStartMessages, expiredStartMessage)
				store.SavePendingStartMessages(
					expiredStartMessage,
				)

				stopMessage := models.NewPendingStopMessage(time.Unix(100, 0), 30, 0, a.AppGuid, a.AppVersion, a.InstanceAtIndex(1).InstanceGuid)
				store.SavePendingStopMessages(
					stopMessage,
				)
			}

			timeProvider.TimeToProvide = time.Unix(130, 0)
			err := sender.Send()
			Ω(err).ShouldNot(HaveOccured())
		})

		It("should limit the number of start messages that it sends", func() {
			remainingStartMessages, _ := store.GetPendingStartMessages()
			Ω(remainingStartMessages).Should(HaveLen(20))
			Ω(messageBus.PublishedMessages["hm9000.start"]).Should(HaveLen(20))

			for _, remainingStartMessage := range remainingStartMessages {
				Ω(validStartMessages).Should(ContainElement(remainingStartMessage))
				Ω(remainingStartMessage.Priority).Should(BeNumerically("<=", 0.5))
			}
		})

		It("should delete all the invalid start messages", func() {
			remainingStartMessages, _ := store.GetPendingStartMessages()
			for _, invalidStartMessage := range invalidStartMessages {
				Ω(remainingStartMessages).ShouldNot(ContainElement(invalidStartMessage))
			}
		})

		It("should delete all the expired start messages", func() {
			remainingStartMessages, _ := store.GetPendingStartMessages()
			for _, expiredStartMessage := range expiredStartMessages {
				Ω(remainingStartMessages).ShouldNot(ContainElement(expiredStartMessage))
			}
		})

		It("should send all the stop messages, as they are cheap to handle", func() {
			remainingStopMessages, _ := store.GetPendingStopMessages()
			Ω(remainingStopMessages).Should(BeEmpty())
			Ω(messageBus.PublishedMessages["hm9000.stop"]).Should(HaveLen(40))
		})
	})
})
