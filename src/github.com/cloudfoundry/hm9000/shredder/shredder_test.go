package shredder_test

import (
	"github.com/cloudfoundry/hm9000/config"
	. "github.com/cloudfoundry/hm9000/shredder"
	storepackage "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	"github.com/cloudfoundry/hm9000/testhelpers/fakestoreadapter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Shredder", func() {
	var (
		shredder     *Shredder
		storeAdapter *fakestoreadapter.FakeStoreAdapter
	)

	BeforeEach(func() {
		storeAdapter = fakestoreadapter.New()
		conf, _ := config.DefaultConfig()
		conf.StoreSchemaVersion = 2
		store := storepackage.NewStore(conf, storeAdapter, fakelogger.NewFakeLogger())
		shredder = New(store)

		storeAdapter.Set([]storeadapter.StoreNode{
			{Key: "/v2/pokemon/geodude", Value: []byte{}},
			{Key: "/v2/deep-pokemon/abra/kadabra/alakazam", Value: []byte{}},
			{Key: "/v2/pokemonCount", Value: []byte("151")},
			{Key: "/v1/nuke/me/cause/im/an/old/version", Value: []byte("abc")},
			{Key: "/v3/leave/me/alone/since/im/a/new/version", Value: []byte("abc")},
			{Key: "/nuke/me/cause/im/not/versioned", Value: []byte("abc")},
		})

		storeAdapter.Delete("/v2/pokemon/geodude", "/v2/deep-pokemon/abra/kadabra/alakazam")
		err := shredder.Shred()
		Ω(err).ShouldNot(HaveOccured())
	})

	It("should delete empty directories", func() {
		_, err := storeAdapter.Get("/v2/pokemon")
		Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))

		_, err = storeAdapter.Get("/v2/deep-pokemon")
		Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))

		_, err = storeAdapter.Get("/v2/pokemonCount")
		Ω(err).ShouldNot(HaveOccured())
	})

	It("should delete everything underneath older versions", func() {
		_, err := storeAdapter.Get("/v1/nuke/me/cause/im/an/old/version")
		Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
	})

	It("should delete everything that is not versioned", func() {
		_, err := storeAdapter.Get("/nuke/me/cause/im/not/versioned")
		Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
	})

	It("should not delete newer versions", func() {
		_, err := storeAdapter.Get("/v3/leave/me/alone/since/im/a/new/version")
		Ω(err).ShouldNot(HaveOccured())
	})
})
