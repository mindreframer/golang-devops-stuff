package store_test

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/workerpool"
	. "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Metrics", func() {
	var (
		store        Store
		storeAdapter storeadapter.StoreAdapter
		conf         *config.Config
	)

	conf, _ = config.DefaultConfig()

	BeforeEach(func() {
		storeAdapter = storeadapter.NewETCDStoreAdapter(etcdRunner.NodeURLS(), workerpool.NewWorkerPool(conf.StoreMaxConcurrentRequests))
		err := storeAdapter.Connect()
		Ω(err).ShouldNot(HaveOccured())

		store = NewStore(conf, storeAdapter, fakelogger.NewFakeLogger())
	})

	Describe("Getting and setting a metric", func() {
		BeforeEach(func() {
			err := store.SaveMetric("sprockets", 17)
			Ω(err).ShouldNot(HaveOccured())
		})

		It("should store the metric under /metrics", func() {
			_, err := storeAdapter.Get("/v1/metrics/sprockets")
			Ω(err).ShouldNot(HaveOccured())
		})

		Context("when the metric is present", func() {
			It("should return the stored value and no error", func() {
				value, err := store.GetMetric("sprockets")
				Ω(err).ShouldNot(HaveOccured())
				Ω(value).Should(BeNumerically("==", 17))
			})

			Context("and it is overwritten", func() {
				BeforeEach(func() {
					err := store.SaveMetric("sprockets", 23.5)
					Ω(err).ShouldNot(HaveOccured())
				})

				It("should return the new value", func() {
					value, err := store.GetMetric("sprockets")
					Ω(err).ShouldNot(HaveOccured())
					Ω(value).Should(BeNumerically("==", 23.5))
				})
			})
		})

		Context("when the metric is not present", func() {
			It("should return -1 and an error", func() {
				value, err := store.GetMetric("nonexistent")
				Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
				Ω(value).Should(BeNumerically("==", -1))
			})
		})
	})
})
