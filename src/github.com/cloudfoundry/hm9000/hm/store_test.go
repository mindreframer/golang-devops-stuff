package hm_test

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/workerpool"
	. "github.com/cloudfoundry/hm9000/hm"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Store", func() {
	var (
		etcdStoreAdapter storeadapter.StoreAdapter
		nodes            []storeadapter.StoreNode
		conf             config.Config
	)

	BeforeEach(func() {
		conf, _ = config.DefaultConfig()
		etcdStoreAdapter = storeadapter.NewETCDStoreAdapter(etcdRunner.NodeURLS(), workerpool.NewWorkerPool(conf.StoreMaxConcurrentRequests))
		err := etcdStoreAdapter.Connect()
		Ω(err).ShouldNot(HaveOccured())

		nodes = []storeadapter.StoreNode{
			{Key: "/desired-fresh", Value: []byte("123"), TTL: 0},
			{Key: "/actual-fresh", Value: []byte("456"), TTL: 0},
			{Key: "/desired/guid1", Value: []byte("guid1"), TTL: 0},
			{Key: "/desired/guid2", Value: []byte("guid2"), TTL: 0},
			{Key: "/menu/oj", Value: []byte("sweet"), TTL: 0},
			{Key: "/menu/breakfast/pancakes", Value: []byte("tasty"), TTL: 0},
			{Key: "/menu/breakfast/waffles", Value: []byte("delish"), TTL: 0},
		}
		etcdStoreAdapter.Set(nodes)
	})

	Describe("Clear", func() {
		It("deletes all entries from store", func() {
			conf.StoreURLs = etcdRunner.NodeURLS()
			Clear(fakelogger.NewFakeLogger(), conf)
			for _, node := range nodes {
				_, err := etcdStoreAdapter.Get(node.Key)
				Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
			}
		})
	})
})
