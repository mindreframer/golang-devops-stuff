package store_test

import (
	. "github.com/cloudfoundry/hm9000/store"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
)

var _ = Describe("Actual State", func() {
	var (
		store       Store
		etcdAdapter storeadapter.StoreAdapter
		conf        config.Config
		heartbeat1  models.InstanceHeartbeat
		heartbeat2  models.InstanceHeartbeat
		heartbeat3  models.InstanceHeartbeat
		app         appfixture.AppFixture
	)

	BeforeEach(func() {
		var err error
		conf, err = config.DefaultConfig()
		Ω(err).ShouldNot(HaveOccured())
		etcdAdapter = storeadapter.NewETCDStoreAdapter(etcdRunner.NodeURLS(), conf.StoreMaxConcurrentRequests)
		err = etcdAdapter.Connect()
		Ω(err).ShouldNot(HaveOccured())

		app = appfixture.NewAppFixture()
		heartbeat1 = app.InstanceAtIndex(0).Heartbeat()
		heartbeat2 = app.InstanceAtIndex(1).Heartbeat()
		heartbeat3 = app.InstanceAtIndex(2).Heartbeat()

		store = NewStore(conf, etcdAdapter, fakelogger.NewFakeLogger())
	})

	AfterEach(func() {
		etcdAdapter.Disconnect()
	})

	Describe("Saving actual state ", func() {
		BeforeEach(func() {
			err := store.SaveActualState(heartbeat1, heartbeat2)
			Ω(err).ShouldNot(HaveOccured())
		})

		It("can stores the passed in actual state", func() {
			node, err := etcdAdapter.ListRecursively("/apps/" + app.AppGuid + "-" + app.AppVersion + "/actual")

			Ω(err).ShouldNot(HaveOccured())
			Ω(node.ChildNodes).Should(HaveLen(2))
			Ω(node.ChildNodes).Should(ContainElement(storeadapter.StoreNode{
				Key:   "/apps/" + app.AppGuid + "-" + app.AppVersion + "/actual/" + heartbeat1.InstanceGuid,
				Value: heartbeat1.ToJSON(),
				TTL:   conf.HeartbeatTTL() - 1,
			}))
			Ω(node.ChildNodes).Should(ContainElement(storeadapter.StoreNode{
				Key:   "/apps/" + app.AppGuid + "-" + app.AppVersion + "/actual/" + heartbeat2.InstanceGuid,
				Value: heartbeat2.ToJSON(),
				TTL:   conf.HeartbeatTTL() - 1,
			}))
		})
	})
})
