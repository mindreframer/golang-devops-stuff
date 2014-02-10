package store_test

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/models"
	. "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	. "github.com/cloudfoundry/hm9000/testhelpers/custommatchers"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/etcdstoreadapter"
	"github.com/cloudfoundry/storeadapter/workerpool"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Apps", func() {
	var (
		store        Store
		storeAdapter storeadapter.StoreAdapter
		conf         *config.Config

		dea        appfixture.DeaFixture
		app1       appfixture.AppFixture
		app2       appfixture.AppFixture
		app3       appfixture.AppFixture
		app4       appfixture.AppFixture
		crashCount []models.CrashCount
	)

	conf, _ = config.DefaultConfig()

	BeforeEach(func() {
		storeAdapter = etcdstoreadapter.NewETCDStoreAdapter(etcdRunner.NodeURLS(), workerpool.NewWorkerPool(conf.StoreMaxConcurrentRequests))
		err := storeAdapter.Connect()
		Ω(err).ShouldNot(HaveOccurred())

		store = NewStore(conf, storeAdapter, fakelogger.NewFakeLogger())

		dea = appfixture.NewDeaFixture()
		app1 = dea.GetApp(0)
		app2 = dea.GetApp(1)
		app3 = dea.GetApp(2)
		app4 = dea.GetApp(3)

		actualState := []models.InstanceHeartbeat{
			app1.InstanceAtIndex(0).Heartbeat(),
			app1.InstanceAtIndex(1).Heartbeat(),
			app1.InstanceAtIndex(2).Heartbeat(),
			app2.InstanceAtIndex(0).Heartbeat(),
		}

		desiredState := []models.DesiredAppState{
			app1.DesiredState(1),
			app3.DesiredState(1),
		}

		crashCount = []models.CrashCount{
			{
				AppGuid:       app1.AppGuid,
				AppVersion:    app1.AppVersion,
				InstanceIndex: 1,
				CrashCount:    12,
			},
			{
				AppGuid:       app1.AppGuid,
				AppVersion:    app1.AppVersion,
				InstanceIndex: 2,
				CrashCount:    17,
			},
			{
				AppGuid:       app2.AppGuid,
				AppVersion:    app2.AppVersion,
				InstanceIndex: 0,
				CrashCount:    3,
			},
			{
				AppGuid:       app4.AppGuid,
				AppVersion:    app4.AppVersion,
				InstanceIndex: 1,
				CrashCount:    8,
			},
		}

		store.SyncHeartbeats(dea.HeartbeatWith(actualState...))
		store.SyncDesiredState(desiredState...)
		store.SaveCrashCounts(crashCount...)
	})

	Describe("AppKey", func() {
		It("should concatenate the app guid and app version appropriately", func() {
			key := store.AppKey("abc", "123")
			Ω(key).Should(Equal("abc,123"))
		})
	})

	Describe("GetApps", func() {
		Context("when all is well", func() {
			It("should build and return the set of apps", func() {
				apps, err := store.GetApps()
				Ω(err).ShouldNot(HaveOccurred())

				Ω(apps).Should(HaveLen(3))
				Ω(apps).Should(HaveKey(app1.AppGuid + "," + app1.AppVersion))
				Ω(apps).Should(HaveKey(app2.AppGuid + "," + app2.AppVersion))
				Ω(apps).Should(HaveKey(app3.AppGuid + "," + app3.AppVersion))

				a1 := apps[app1.AppGuid+","+app1.AppVersion]
				Ω(a1.Desired).Should(EqualDesiredState(app1.DesiredState(1)))
				Ω(a1.InstanceHeartbeats).Should(HaveLen(3))
				Ω(a1.InstanceHeartbeats).Should(ContainElement(app1.InstanceAtIndex(0).Heartbeat()))
				Ω(a1.InstanceHeartbeats).Should(ContainElement(app1.InstanceAtIndex(1).Heartbeat()))
				Ω(a1.InstanceHeartbeats).Should(ContainElement(app1.InstanceAtIndex(2).Heartbeat()))
				Ω(a1.CrashCounts[1]).Should(Equal(crashCount[0]))
				Ω(a1.CrashCounts[2]).Should(Equal(crashCount[1]))

				a2 := apps[app2.AppGuid+","+app2.AppVersion]
				Ω(a2.Desired).Should(BeZero())
				Ω(a2.InstanceHeartbeats).Should(HaveLen(1))
				Ω(a2.InstanceHeartbeats).Should(ContainElement(app2.InstanceAtIndex(0).Heartbeat()))
				Ω(a2.CrashCounts[0]).Should(Equal(crashCount[2]))

				a3 := apps[app3.AppGuid+","+app3.AppVersion]
				Ω(a3.Desired).Should(EqualDesiredState(app3.DesiredState(1)))
				Ω(a3.InstanceHeartbeats).Should(HaveLen(0))
				Ω(a3.CrashCounts).Should(BeEmpty())
			})
		})

		Context("when there is an empty app directory", func() {
			It("should ignore that app directory", func() {
				storeAdapter.SetMulti([]storeadapter.StoreNode{{
					Key:   "/hm/v1/apps/actual/foo-bar",
					Value: []byte("foo"),
				}})

				apps, err := store.GetApps()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(apps).Should(HaveLen(3))
			})
		})
	})

	Describe("GetApp", func() {
		Context("when there are no store errors", func() {
			Context("when the app has desired and actual state", func() {
				It("should return the app", func() {
					app, err := store.GetApp(app1.AppGuid, app1.AppVersion)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(app.Desired).Should(EqualDesiredState(app1.DesiredState(1)))
					Ω(app.InstanceHeartbeats).Should(HaveLen(3))
					Ω(app.InstanceHeartbeats).Should(ContainElement(app1.InstanceAtIndex(0).Heartbeat()))
					Ω(app.InstanceHeartbeats).Should(ContainElement(app1.InstanceAtIndex(1).Heartbeat()))
					Ω(app.InstanceHeartbeats).Should(ContainElement(app1.InstanceAtIndex(2).Heartbeat()))
					Ω(app.CrashCounts[1]).Should(Equal(crashCount[0]))
					Ω(app.CrashCounts[2]).Should(Equal(crashCount[1]))
				})
			})

			Context("when the app has desired state only", func() {
				It("should return the app", func() {
					app, err := store.GetApp(app3.AppGuid, app3.AppVersion)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(app.Desired).Should(EqualDesiredState(app3.DesiredState(1)))
					Ω(app.InstanceHeartbeats).Should(BeEmpty())
					Ω(app.CrashCounts).Should(BeEmpty())
				})
			})

			Context("when the app has actual state only", func() {
				It("should return the app", func() {
					app, err := store.GetApp(app2.AppGuid, app2.AppVersion)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(app.Desired).Should(BeZero())
					Ω(app.InstanceHeartbeats).Should(HaveLen(1))
					Ω(app.InstanceHeartbeats).Should(ContainElement(app2.InstanceAtIndex(0).Heartbeat()))
					Ω(app.CrashCounts[0]).Should(Equal(crashCount[2]))
				})
			})

			Context("when the app has crash counts only", func() {
				It("should return the app not found error", func() {
					app, err := store.GetApp(app4.AppGuid, app4.AppVersion)
					Ω(err).Should(Equal(AppNotFoundError))
					Ω(app).Should(BeZero())
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
					storeAdapter.SetMulti([]storeadapter.StoreNode{{
						Key:   "/hm/v1/apps/actual/foo-bar/baz",
						Value: []byte("foo"),
					}})

					storeAdapter.Delete("/hm/v1/apps/actual/foo-bar/baz")

					app, err := store.GetApp("foo", "bar")
					Ω(err).Should(Equal(AppNotFoundError))
					Ω(app).Should(BeZero())
				})
			})

		})
	})
})
