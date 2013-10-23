package store_test

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/models"
	. "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	. "github.com/cloudfoundry/hm9000/testhelpers/custommatchers"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Apps", func() {
	var (
		store       Store
		etcdAdapter storeadapter.StoreAdapter
		conf        config.Config

		fixture1   appfixture.AppFixture
		fixture2   appfixture.AppFixture
		fixture3   appfixture.AppFixture
		crashCount []models.CrashCount
	)

	conf, _ = config.DefaultConfig()

	BeforeEach(func() {
		etcdAdapter = storeadapter.NewETCDStoreAdapter(etcdRunner.NodeURLS(), conf.StoreMaxConcurrentRequests)
		err := etcdAdapter.Connect()
		Ω(err).ShouldNot(HaveOccured())

		store = NewStore(conf, etcdAdapter, fakelogger.NewFakeLogger())

		fixture1 = appfixture.NewAppFixture()
		fixture2 = appfixture.NewAppFixture()
		fixture3 = appfixture.NewAppFixture()

		actualState := []models.InstanceHeartbeat{
			fixture1.InstanceAtIndex(0).Heartbeat(),
			fixture1.InstanceAtIndex(1).Heartbeat(),
			fixture1.InstanceAtIndex(2).Heartbeat(),
			fixture2.InstanceAtIndex(0).Heartbeat(),
		}

		desiredState := []models.DesiredAppState{
			fixture1.DesiredState(1),
			fixture3.DesiredState(1),
		}

		crashCount = []models.CrashCount{
			models.CrashCount{
				AppGuid:       fixture1.AppGuid,
				AppVersion:    fixture1.AppVersion,
				InstanceIndex: 1,
				CrashCount:    12,
			},
			models.CrashCount{
				AppGuid:       fixture1.AppGuid,
				AppVersion:    fixture1.AppVersion,
				InstanceIndex: 2,
				CrashCount:    17,
			},
			models.CrashCount{
				AppGuid:       fixture2.AppGuid,
				AppVersion:    fixture2.AppVersion,
				InstanceIndex: 0,
				CrashCount:    3,
			},
		}

		store.SaveActualState(actualState...)
		store.SaveDesiredState(desiredState...)
		store.SaveCrashCounts(crashCount...)
	})

	Describe("AppKey", func() {
		It("should concatenate the app guid and app version appropriately", func() {
			key := store.AppKey("abc", "123")
			Ω(key).Should(Equal("abc-123"))
		})
	})

	Describe("GetApps", func() {
		Context("when all is well", func() {
			It("should build and return the set of apps", func() {
				apps, err := store.GetApps()
				Ω(err).ShouldNot(HaveOccured())

				Ω(apps).Should(HaveLen(3))
				Ω(apps).Should(HaveKey(fixture1.AppGuid + "-" + fixture1.AppVersion))
				Ω(apps).Should(HaveKey(fixture2.AppGuid + "-" + fixture2.AppVersion))
				Ω(apps).Should(HaveKey(fixture3.AppGuid + "-" + fixture3.AppVersion))

				a1 := apps[fixture1.AppGuid+"-"+fixture1.AppVersion]
				Ω(a1.Desired).Should(EqualDesiredState(fixture1.DesiredState(1)))
				Ω(a1.InstanceHeartbeats).Should(HaveLen(3))
				Ω(a1.InstanceHeartbeats).Should(ContainElement(fixture1.InstanceAtIndex(0).Heartbeat()))
				Ω(a1.InstanceHeartbeats).Should(ContainElement(fixture1.InstanceAtIndex(1).Heartbeat()))
				Ω(a1.InstanceHeartbeats).Should(ContainElement(fixture1.InstanceAtIndex(2).Heartbeat()))
				Ω(a1.CrashCounts[1]).Should(Equal(crashCount[0]))
				Ω(a1.CrashCounts[2]).Should(Equal(crashCount[1]))

				a2 := apps[fixture2.AppGuid+"-"+fixture2.AppVersion]
				Ω(a2.Desired).Should(BeZero())
				Ω(a2.InstanceHeartbeats).Should(HaveLen(1))
				Ω(a2.InstanceHeartbeats).Should(ContainElement(fixture2.InstanceAtIndex(0).Heartbeat()))
				Ω(a2.CrashCounts[0]).Should(Equal(crashCount[2]))

				a3 := apps[fixture3.AppGuid+"-"+fixture3.AppVersion]
				Ω(a3.Desired).Should(EqualDesiredState(fixture3.DesiredState(1)))
				Ω(a3.InstanceHeartbeats).Should(HaveLen(0))
				Ω(a3.CrashCounts).Should(BeEmpty())
			})
		})

		Context("when there is an empty app directory", func() {
			It("should ignore that app directory", func() {
				etcdAdapter.Set([]storeadapter.StoreNode{storeadapter.StoreNode{
					Key:   "/apps/foo-bar/empty",
					Value: []byte("foo"),
				}})

				apps, err := store.GetApps()
				Ω(err).ShouldNot(HaveOccured())
				Ω(apps).Should(HaveLen(3))
			})
		})
	})

	Describe("GetApp", func() {
		Context("when there are no store errors", func() {
			Context("when the app is found", func() {
				It("should return the app", func() {
					app, err := store.GetApp(fixture1.AppGuid, fixture1.AppVersion)
					Ω(err).ShouldNot(HaveOccured())
					Ω(app.Desired).Should(EqualDesiredState(fixture1.DesiredState(1)))
					Ω(app.InstanceHeartbeats).Should(HaveLen(3))
					Ω(app.InstanceHeartbeats).Should(ContainElement(fixture1.InstanceAtIndex(0).Heartbeat()))
					Ω(app.InstanceHeartbeats).Should(ContainElement(fixture1.InstanceAtIndex(1).Heartbeat()))
					Ω(app.InstanceHeartbeats).Should(ContainElement(fixture1.InstanceAtIndex(2).Heartbeat()))
					Ω(app.CrashCounts[1]).Should(Equal(crashCount[0]))
					Ω(app.CrashCounts[2]).Should(Equal(crashCount[1]))
				})
			})

			Context("when the app is not found", func() {
				It("should return the app not found error", func() {
					app, err := store.GetApp("Marzipan", "Armadillo")
					Ω(err).Should(Equal(AppNotFoundError))
					Ω(app).Should(BeZero())
				})
			})

			Context("when the app directory is empty", func() {
				It("should return the app not found error", func() {
					etcdAdapter.Set([]storeadapter.StoreNode{storeadapter.StoreNode{
						Key:   "/apps/foo-bar/empty",
						Value: []byte("foo"),
					}})

					app, err := store.GetApp("foo", "bar")
					Ω(err).Should(Equal(AppNotFoundError))
					Ω(app).Should(BeZero())
				})
			})

		})
	})
})
