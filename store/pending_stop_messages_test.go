package store_test

import (
	"github.com/cloudfoundry/hm9000/config"
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
		store       Store
		etcdAdapter storeadapter.StoreAdapter
		conf        config.Config
		message1    models.PendingStopMessage
		message2    models.PendingStopMessage
		message3    models.PendingStopMessage
	)

	BeforeEach(func() {
		var err error
		conf, err = config.DefaultConfig()
		Ω(err).ShouldNot(HaveOccured())
		etcdAdapter = storeadapter.NewETCDStoreAdapter(etcdRunner.NodeURLS(), conf.StoreMaxConcurrentRequests)
		err = etcdAdapter.Connect()
		Ω(err).ShouldNot(HaveOccured())

		message1 = models.NewPendingStopMessage(time.Unix(100, 0), 10, 4, "ABC", "123", "XYZ")
		message2 = models.NewPendingStopMessage(time.Unix(100, 0), 10, 4, "DEF", "456", "ALPHA")
		message3 = models.NewPendingStopMessage(time.Unix(100, 0), 10, 4, "GHI", "789", "BETA")

		store = NewStore(conf, etcdAdapter, fakelogger.NewFakeLogger())
	})

	AfterEach(func() {
		etcdAdapter.Disconnect()
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
			node, err := etcdAdapter.ListRecursively("/stop")
			Ω(err).ShouldNot(HaveOccured())
			Ω(node.ChildNodes).Should(HaveLen(2))
			Ω(node.ChildNodes).Should(ContainElement(storeadapter.StoreNode{
				Key:   "/stop/" + message1.StoreKey(),
				Value: message1.ToJSON(),
				TTL:   0,
			}))
			Ω(node.ChildNodes).Should(ContainElement(storeadapter.StoreNode{
				Key:   "/stop/" + message2.StoreKey(),
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
				_, err := etcdAdapter.ListRecursively("/stop")
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

		Context("When the stop message is present", func() {
			It("can delete the stop message (and only cares about the relevant fields)", func() {
				toDelete := []models.PendingStopMessage{
					models.NewPendingStopMessage(time.Time{}, 0, 0, "", "", message1.InstanceGuid),
					models.NewPendingStopMessage(time.Time{}, 0, 0, "", "", message3.InstanceGuid),
				}
				err := store.DeletePendingStopMessages(toDelete...)
				Ω(err).ShouldNot(HaveOccured())

				desired, err := store.GetPendingStopMessages()
				Ω(err).ShouldNot(HaveOccured())
				Ω(desired).Should(HaveLen(1))
				Ω(desired).Should(ContainElement(message2))
			})
		})

		Context("When the desired message key is not present", func() {
			It("returns an error, but does leave things in a broken state... for now...", func() {
				toDelete := []models.PendingStopMessage{
					models.NewPendingStopMessage(time.Time{}, 0, 0, "", "", message1.InstanceGuid),
					models.NewPendingStopMessage(time.Time{}, 0, 0, "", "", "floobedey"),
					models.NewPendingStopMessage(time.Time{}, 0, 0, "", "", message3.InstanceGuid),
				}
				err := store.DeletePendingStopMessages(toDelete...)
				Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))

				stop, err := store.GetPendingStopMessages()
				Ω(err).ShouldNot(HaveOccured())
				Ω(stop).Should(HaveLen(2))
				Ω(stop).Should(ContainElement(message2))
				Ω(stop).Should(ContainElement(message3))
			})
		})
	})
})
