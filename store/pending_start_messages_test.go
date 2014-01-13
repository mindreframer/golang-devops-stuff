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

var _ = Describe("Storing PendingStartMessages", func() {
	var (
		store        Store
		storeAdapter storeadapter.StoreAdapter
		conf         *config.Config
		message1     models.PendingStartMessage
		message2     models.PendingStartMessage
		message3     models.PendingStartMessage
	)

	BeforeEach(func() {
		var err error
		conf, err = config.DefaultConfig()
		Ω(err).ShouldNot(HaveOccurred())
		storeAdapter = storeadapter.NewETCDStoreAdapter(etcdRunner.NodeURLS(), workerpool.NewWorkerPool(conf.StoreMaxConcurrentRequests))
		err = storeAdapter.Connect()
		Ω(err).ShouldNot(HaveOccurred())

		message1 = models.NewPendingStartMessage(time.Unix(100, 0), 10, 4, "ABC", "123", 1, 1.0, models.PendingStartMessageReasonInvalid)
		message2 = models.NewPendingStartMessage(time.Unix(100, 0), 10, 4, "DEF", "123", 1, 1.0, models.PendingStartMessageReasonInvalid)
		message3 = models.NewPendingStartMessage(time.Unix(100, 0), 10, 4, "ABC", "456", 1, 1.0, models.PendingStartMessageReasonInvalid)

		store = NewStore(conf, storeAdapter, fakelogger.NewFakeLogger())
	})

	AfterEach(func() {
		storeAdapter.Disconnect()
	})

	Describe("Saving start messages", func() {
		BeforeEach(func() {
			err := store.SavePendingStartMessages(
				message1,
				message2,
			)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("stores the passed in start messages", func() {
			node, err := storeAdapter.ListRecursively("/hm/v1/start")
			Ω(err).ShouldNot(HaveOccurred())
			Ω(node.ChildNodes).Should(HaveLen(2))
			Ω(node.ChildNodes).Should(ContainElement(storeadapter.StoreNode{
				Key:   "/hm/v1/start/" + message1.StoreKey(),
				Value: message1.ToJSON(),
				TTL:   0,
			}))
			Ω(node.ChildNodes).Should(ContainElement(storeadapter.StoreNode{
				Key:   "/hm/v1/start/" + message2.StoreKey(),
				Value: message2.ToJSON(),
				TTL:   0,
			}))
		})
	})

	Describe("Fetching start message", func() {
		Context("When the start message is present", func() {
			BeforeEach(func() {
				err := store.SavePendingStartMessages(
					message1,
					message2,
				)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("can fetch the start message", func() {
				desired, err := store.GetPendingStartMessages()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(desired).Should(HaveLen(2))
				Ω(desired).Should(ContainElement(message1))
				Ω(desired).Should(ContainElement(message2))
			})
		})

		Context("when the start message is empty", func() {
			BeforeEach(func() {
				hb := message1
				err := store.SavePendingStartMessages(hb)
				Ω(err).ShouldNot(HaveOccurred())
				err = store.DeletePendingStartMessages(hb)
				Ω(err).ShouldNot(HaveOccurred())
			})

			It("returns an empty array", func() {
				start, err := store.GetPendingStartMessages()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(start).Should(BeEmpty())
			})
		})

		Context("When the start message key is missing", func() {
			BeforeEach(func() {
				_, err := storeAdapter.ListRecursively("/hm/v1/start")
				Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
			})

			It("returns an empty array and no error", func() {
				start, err := store.GetPendingStartMessages()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(start).Should(BeEmpty())
			})
		})
	})

	Describe("Deleting start message", func() {
		BeforeEach(func() {
			err := store.SavePendingStartMessages(
				message1,
				message2,
				message3,
			)
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("can deletes start messages", func() {
			toDelete := []models.PendingStartMessage{
				models.NewPendingStartMessage(time.Time{}, 0, 0, message1.AppGuid, message1.AppVersion, message1.IndexToStart, 0, models.PendingStartMessageReasonInvalid),
				models.NewPendingStartMessage(time.Time{}, 0, 0, message3.AppGuid, message3.AppVersion, message3.IndexToStart, 0, models.PendingStartMessageReasonInvalid),
			}
			err := store.DeletePendingStartMessages(toDelete...)
			Ω(err).ShouldNot(HaveOccurred())

			desired, err := store.GetPendingStartMessages()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(desired).Should(HaveLen(1))
			Ω(desired).Should(ContainElement(message2))
		})
	})
})
