package store_test

import (
	. "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/storeadapter/workerpool"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/etcdstoreadapter"
)

var _ = Describe("Compact", func() {
	var (
		store        Store
		storeAdapter storeadapter.StoreAdapter
		conf         *config.Config
	)

	BeforeEach(func() {
		var err error
		conf, err = config.DefaultConfig()
		conf.StoreSchemaVersion = 17
		Ω(err).ShouldNot(HaveOccurred())
		storeAdapter = etcdstoreadapter.NewETCDStoreAdapter(etcdRunner.NodeURLS(), workerpool.NewWorkerPool(conf.StoreMaxConcurrentRequests))
		err = storeAdapter.Connect()
		Ω(err).ShouldNot(HaveOccurred())
		store = NewStore(conf, storeAdapter, fakelogger.NewFakeLogger())
	})

	Describe("Deleting old schema version", func() {
		BeforeEach(func() {
			storeAdapter.SetMulti([]storeadapter.StoreNode{
				{Key: "/hm/v3/delete/me", Value: []byte("abc")},
				{Key: "/hm/v16/delete/me", Value: []byte("abc")},
				{Key: "/hm/v17/leave/me/alone", Value: []byte("abc")},
				{Key: "/hm/v17/leave/me/v1/alone", Value: []byte("abc")},
				{Key: "/hm/v18/leave/me/alone", Value: []byte("abc")},
				{Key: "/hm/delete/me", Value: []byte("abc")},
				{Key: "/hm/v1ola/delete/me", Value: []byte("abc")},
				{Key: "/hm/delete/me/too", Value: []byte("abc")},
				{Key: "/hm/locks/keep", Value: []byte("abc")},
				{Key: "/other/keep", Value: []byte("abc")},
				{Key: "/foo", Value: []byte("abc")},
				{Key: "/v3/keep", Value: []byte("abc")},
			})

			err := store.Compact()
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should delete everything under older versions", func() {
			_, err := storeAdapter.Get("/hm/v3/delete/me")
			Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))

			_, err = storeAdapter.Get("/hm/v16/delete/me")
			Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
		})

		It("should leave the current version alone", func() {
			_, err := storeAdapter.Get("/hm/v17/leave/me/alone")
			Ω(err).ShouldNot(HaveOccurred())

			_, err = storeAdapter.Get("/hm/v17/leave/me/v1/alone")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should leave newer versions alone", func() {
			_, err := storeAdapter.Get("/hm/v18/leave/me/alone")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should leave locks alone", func() {
			_, err := storeAdapter.Get("/hm/locks/keep")
			Ω(err).ShouldNot(HaveOccurred())
		})

		It("should delete anything that's unversioned", func() {
			_, err := storeAdapter.Get("/hm/delete/me")
			Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))

			_, err = storeAdapter.Get("/hm/v1ola/delete/me")
			Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))

			_, err = storeAdapter.Get("/hm/delete/me/too")
			Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
		})

		It("should not touch anything that isn't under the hm namespace", func() {
			_, err := storeAdapter.Get("/other/keep")
			Ω(err).ShouldNot(HaveOccurred())

			_, err = storeAdapter.Get("/foo")
			Ω(err).ShouldNot(HaveOccurred())

			_, err = storeAdapter.Get("/v3/keep")
			Ω(err).ShouldNot(HaveOccurred())
		})
	})

	Describe("Recursively deleting empty directories", func() {
		BeforeEach(func() {
			storeAdapter.SetMulti([]storeadapter.StoreNode{
				{Key: "/hm/v17/pokemon/geodude", Value: []byte("foo")},
				{Key: "/hm/v17/deep-pokemon/abra/kadabra/alakazam", Value: []byte{}},
				{Key: "/hm/v17/pokemonCount", Value: []byte("151")},
			})
		})

		Context("when the node is a directory", func() {
			Context("and it is empty", func() {
				BeforeEach(func() {
					storeAdapter.Delete("/hm/v17/pokemon/geodude")
				})

				It("shreds it mercilessly", func() {
					err := store.Compact()
					Ω(err).ShouldNot(HaveOccurred())

					_, err = storeAdapter.Get("/hm/v17/pokemon")
					Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
				})
			})

			Context("and it is non-empty", func() {
				It("spares it", func() {
					err := store.Compact()
					Ω(err).ShouldNot(HaveOccurred())

					_, err = storeAdapter.Get("/hm/v17/pokemon/geodude")
					Ω(err).ShouldNot(HaveOccurred())
				})

				Context("but all of its children are empty", func() {
					BeforeEach(func() {
						storeAdapter.Delete("/hm/v17/deep-pokemon/abra/kadabra/alakazam")
					})

					It("shreds it mercilessly", func() {
						err := store.Compact()
						Ω(err).ShouldNot(HaveOccurred())

						_, err = storeAdapter.Get("/hm/v17/deep-pokemon/abra/kadabra")
						Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))

						_, err = storeAdapter.Get("/hm/v17/deep-pokemon/abra")
						Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
					})
				})
			})
		})

		Context("when the node is NOT a directory", func() {
			It("spares it", func() {
				err := store.Compact()
				Ω(err).ShouldNot(HaveOccurred())

				_, err = storeAdapter.Get("/hm/v17/pokemonCount")
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

	})
})
