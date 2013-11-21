package store_test

import (
	"github.com/cloudfoundry/hm9000/helpers/workerpool"
	. "github.com/cloudfoundry/hm9000/store"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
)

var _ = Describe("Compact", func() {
	var (
		store        Store
		storeAdapter storeadapter.StoreAdapter
		conf         config.Config
	)

	BeforeEach(func() {
		var err error
		conf, err = config.DefaultConfig()
		conf.StoreSchemaVersion = 17
		Ω(err).ShouldNot(HaveOccured())
		storeAdapter = storeadapter.NewETCDStoreAdapter(etcdRunner.NodeURLS(), workerpool.NewWorkerPool(conf.StoreMaxConcurrentRequests))
		err = storeAdapter.Connect()
		Ω(err).ShouldNot(HaveOccured())
		store = NewStore(conf, storeAdapter, fakelogger.NewFakeLogger())
	})

	Describe("Deleting old schema version", func() {
		BeforeEach(func() {
			storeAdapter.Set([]storeadapter.StoreNode{
				{Key: "/v3/delete/me", Value: []byte("abc")},
				{Key: "/v16/delete/me", Value: []byte("abc")},
				{Key: "/v17/leave/me/alone", Value: []byte("abc")},
				{Key: "/v17/leave/me/v1/alone", Value: []byte("abc")},
				{Key: "/v18/leave/me/alone", Value: []byte("abc")},
				{Key: "/delete/me", Value: []byte("abc")},
				{Key: "/v1ola/delete/me", Value: []byte("abc")},
				{Key: "/delete/me/too", Value: []byte("abc")},
			})

			err := store.Compact()
			Ω(err).ShouldNot(HaveOccured())
		})

		It("should delete everything under older versions", func() {
			_, err := storeAdapter.Get("/v3/delete/me")
			Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))

			_, err = storeAdapter.Get("/v16/delete/me")
			Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
		})

		It("should leave the current version alone", func() {
			_, err := storeAdapter.Get("/v17/leave/me/alone")
			Ω(err).ShouldNot(HaveOccured())

			_, err = storeAdapter.Get("/v17/leave/me/v1/alone")
			Ω(err).ShouldNot(HaveOccured())
		})

		It("should leave newer versions alone", func() {
			_, err := storeAdapter.Get("/v18/leave/me/alone")
			Ω(err).ShouldNot(HaveOccured())
		})

		It("should delete anything that's unversioned", func() {
			_, err := storeAdapter.Get("/delete/me")
			Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))

			_, err = storeAdapter.Get("/v1ola/delete/me")
			Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))

			_, err = storeAdapter.Get("/delete/me/too")
			Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
		})
	})

	Describe("Recursively deleting empty directories", func() {
		BeforeEach(func() {
			storeAdapter.Set([]storeadapter.StoreNode{
				{Key: "/v17/pokemon/geodude", Value: []byte("foo")},
				{Key: "/v17/deep-pokemon/abra/kadabra/alakazam", Value: []byte{}},
				{Key: "/v17/pokemonCount", Value: []byte("151")},
			})
		})

		Context("when the node is a directory", func() {
			Context("and it is empty", func() {
				BeforeEach(func() {
					storeAdapter.Delete("/v17/pokemon/geodude")
				})

				It("shreds it mercilessly", func() {
					err := store.Compact()
					Ω(err).ShouldNot(HaveOccured())

					_, err = storeAdapter.Get("/v17/pokemon")
					Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
				})
			})

			Context("and it is non-empty", func() {
				It("spares it", func() {
					err := store.Compact()
					Ω(err).ShouldNot(HaveOccured())

					_, err = storeAdapter.Get("/v17/pokemon/geodude")
					Ω(err).ShouldNot(HaveOccured())
				})

				Context("but all of its children are empty", func() {
					BeforeEach(func() {
						storeAdapter.Delete("/v17/deep-pokemon/abra/kadabra/alakazam")
					})

					It("shreds it mercilessly", func() {
						err := store.Compact()
						Ω(err).ShouldNot(HaveOccured())

						_, err = storeAdapter.Get("/v17/deep-pokemon/abra/kadabra")
						Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))

						_, err = storeAdapter.Get("/v17/deep-pokemon/abra")
						Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
					})
				})
			})
		})

		Context("when the node is NOT a directory", func() {
			It("spares it", func() {
				err := store.Compact()
				Ω(err).ShouldNot(HaveOccured())

				_, err = storeAdapter.Get("/v17/pokemonCount")
				Ω(err).ShouldNot(HaveOccured())
			})
		})

	})
})
