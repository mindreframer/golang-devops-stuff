package store_test

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/workerpool"
	"github.com/cloudfoundry/hm9000/models"
	. "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Storing PendingStopMessages", func() {
	var (
		store        Store
		storeAdapter storeadapter.StoreAdapter
		conf         *config.Config
		message1     models.PendingStopMessage
		message2     models.PendingStopMessage
		message3     models.PendingStopMessage
	)

	BeforeEach(func() {
		var err error
		conf, err = config.DefaultConfig()
		Ω(err).ShouldNot(HaveOccured())
		storeAdapter = storeadapter.NewETCDStoreAdapter(etcdRunner.NodeURLS(), workerpool.NewWorkerPool(conf.StoreMaxConcurrentRequests))
		err = storeAdapter.Connect()
		Ω(err).ShouldNot(HaveOccured())

		message1 = models.NewPendingStopMessage(time.Unix(100, 0), 10, 4, "ABC", "123", "XYZ", models.PendingStopMessageReasonInvalid)
		message2 = models.NewPendingStopMessage(time.Unix(100, 0), 10, 4, "DEF", "456", "ALPHA", models.PendingStopMessageReasonInvalid)
		message3 = models.NewPendingStopMessage(time.Unix(100, 0), 10, 4, "GHI", "789", "BETA", models.PendingStopMessageReasonInvalid)

		store = NewStore(conf, storeAdapter, fakelogger.NewFakeLogger())
	})

	AfterEach(func() {
		storeAdapter.Disconnect()
	})

	Describe("Saving stop messages", func() {
		BeforeEach(func() {
			err := store.SavePendingStopMessages(
				message1,
				message2,
			)
			Ω(err).ShouldNot(HaveOccured())
		})

		It("stores the passed in stop messages", func() {
			node, err := storeAdapter.ListRecursively("/v1/stop")
			Ω(err).ShouldNot(HaveOccured())
			Ω(node.ChildNodes).Should(HaveLen(2))
			Ω(node.ChildNodes).Should(ContainElement(storeadapter.StoreNode{
				Key:   "/v1/stop/" + message1.StoreKey(),
				Value: message1.ToJSON(),
				TTL:   0,
			}))
			Ω(node.ChildNodes).Should(ContainElement(storeadapter.StoreNode{
				Key:   "/v1/stop/" + message2.StoreKey(),
				Value: message2.ToJSON(),
				TTL:   0,
			}))
		})
	})

	Describe("Fetching stop message", func() {
		Context("When the stop message is present", func() {
			BeforeEach(func() {
				err := store.SavePendingStopMessages(
					message1,
					message2,
				)
				Ω(err).ShouldNot(HaveOccured())
			})

			It("can fetch the stop message", func() {
				desired, err := store.GetPendingStopMessages()
				Ω(err).ShouldNot(HaveOccured())
				Ω(desired).Should(HaveLen(2))
				Ω(desired).Should(ContainElement(message1))
				Ω(desired).Should(ContainElement(message2))
			})
		})

		Context("when the stop message is empty", func() {
			BeforeEach(func() {
				hb := message1
				err := store.SavePendingStopMessages(hb)
				Ω(err).ShouldNot(HaveOccured())
				err = store.DeletePendingStopMessages(hb)
				Ω(err).ShouldNot(HaveOccured())
			})

			It("returns an empty array", func() {
				stop, err := store.GetPendingStopMessages()
				Ω(err).ShouldNot(HaveOccured())
				Ω(stop).Should(BeEmpty())
			})
		})

		Context("When the stop message key is missing", func() {
			BeforeEach(func() {
				_, err := storeAdapter.ListRecursively("/v1/stop")
				Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
			})

			It("returns an empty array and no error", func() {
				stop, err := store.GetPendingStopMessages()
				Ω(err).ShouldNot(HaveOccured())
				Ω(stop).Should(BeEmpty())
			})
		})
	})

	Describe("Deleting stop message", func() {
		BeforeEach(func() {
			err := store.SavePendingStopMessages(
				message1,
				message2,
				message3,
			)
			Ω(err).ShouldNot(HaveOccured())
		})

		It("deletes stop messages (and only cares about the relevant fields)", func() {
			toDelete := []models.PendingStopMessage{
				models.NewPendingStopMessage(time.Time{}, 0, 0, "", "", message1.InstanceGuid, models.PendingStopMessageReasonInvalid),
				models.NewPendingStopMessage(time.Time{}, 0, 0, "", "", message3.InstanceGuid, models.PendingStopMessageReasonInvalid),
			}
			err := store.DeletePendingStopMessages(toDelete...)
			Ω(err).ShouldNot(HaveOccured())

			desired, err := store.GetPendingStopMessages()
			Ω(err).ShouldNot(HaveOccured())
			Ω(desired).Should(HaveLen(1))
			Ω(desired).Should(ContainElement(message2))
		})
	})
})
