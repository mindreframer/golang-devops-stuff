package metricsserver_test

import (
	"errors"
	"github.com/cloudfoundry/hm9000/config"
	. "github.com/cloudfoundry/hm9000/metricsserver"
	"github.com/cloudfoundry/hm9000/models"
	storepackage "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	"github.com/cloudfoundry/hm9000/testhelpers/fakemetricsaccountant"
	"github.com/cloudfoundry/hm9000/testhelpers/fakestoreadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/faketimeprovider"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Metrics Server", func() {
	var (
		store             storepackage.Store
		storeAdapter      *fakestoreadapter.FakeStoreAdapter
		timeProvider      *faketimeprovider.FakeTimeProvider
		metricsServer     *MetricsServer
		metricsAccountant *fakemetricsaccountant.FakeMetricsAccountant
	)

	BeforeEach(func() {
		conf, _ := config.DefaultConfig()
		storeAdapter = fakestoreadapter.New()
		store = storepackage.NewStore(conf, storeAdapter, fakelogger.NewFakeLogger())
		timeProvider = &faketimeprovider.FakeTimeProvider{TimeToProvide: time.Unix(100, 0)}

		metricsAccountant = fakemetricsaccountant.New()

		metricsServer = New(nil, nil, metricsAccountant, fakelogger.NewFakeLogger(), store, timeProvider, conf)
	})

	Describe("message metrics", func() {
		BeforeEach(func() {
			metricsAccountant.GetMetricsMetrics = map[string]float64{
				"foo": 1,
				"bar": 2,
			}
		})

		Context("when the metrics fetch succesfully", func() {
			It("should return them", func() {
				context := metricsServer.Emit()
				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "foo", Value: float64(1.0)}))
				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "bar", Value: float64(2.0)}))
			})
		})

		Context("when the metrics fail to fetch", func() {
			BeforeEach(func() {
				metricsAccountant.GetMetricsError = errors.New("oops")
			})

			It("should not return them", func() {
				context := metricsServer.Emit()
				Ω(context.Metrics).ShouldNot(ContainElement(instrumentation.Metric{Name: "foo", Value: float64(1.0)}))
				Ω(context.Metrics).ShouldNot(ContainElement(instrumentation.Metric{Name: "bar", Value: float64(2.0)}))
			})
		})
	})

	Describe("app metrics", func() {
		It("should have a name", func() {
			context := metricsServer.Emit()
			Ω(context.Name).Should(Equal("HM9000"))
		})

		Context("when the store is not fresh", func() {
			It("should emit -1 for all its metrics", func() {
				context := metricsServer.Emit()
				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: -1}))
				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: -1}))
				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: -1}))
				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: -1}))
				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: -1}))
				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: -1}))
				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: -1}))
				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredApps", Value: -1}))
				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredInstances", Value: -1}))
				Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredAppsPendingStaging", Value: -1}))
			})
		})

		Context("when the store is fresh", func() {
			var dea appfixture.DeaFixture
			var a appfixture.AppFixture

			BeforeEach(func() {
				dea = appfixture.NewDeaFixture()
				a = dea.GetApp(0)
				store.BumpDesiredFreshness(time.Unix(0, 0))
				store.BumpActualFreshness(time.Unix(0, 0))
			})

			Context("when the apps fail to load", func() {
				BeforeEach(func() {
					storeAdapter.ListErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector("apps", errors.New("oops"))
				})

				It("should emit -1 for all its metrics", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: -1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: -1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: -1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: -1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: -1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: -1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: -1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredApps", Value: -1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredInstances", Value: -1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredAppsPendingStaging", Value: -1}))
				})
			})

			Context("When a desired app is pending staging", func() {
				BeforeEach(func() {
					desired := a.DesiredState(3)
					desired.PackageState = models.AppPackageStatePending
					store.SyncDesiredState(desired)
					store.SyncHeartbeats(a.Heartbeat(1))
				})

				It("should have the correct stats", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredAppsPendingStaging", Value: 1}))
				})
			})

			Context("when a desired app has all instances running", func() {
				BeforeEach(func() {
					store.SyncDesiredState(a.DesiredState(3))
					store.SyncHeartbeats(a.Heartbeat(3))
				})

				It("should have the correct stats", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredApps", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredAppsPendingStaging", Value: 0}))
				})
			})

			Context("when a desired app has an instance starting and others running", func() {
				BeforeEach(func() {
					store.SyncDesiredState(a.DesiredState(3))

					startingHB := a.InstanceAtIndex(1).Heartbeat()
					startingHB.State = models.InstanceStateStarting
					store.SyncHeartbeats(dea.HeartbeatWith(
						a.InstanceAtIndex(0).Heartbeat(),
						startingHB,
						a.InstanceAtIndex(2).Heartbeat(),
					))
				})

				It("should have the correct stats", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredApps", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredAppsPendingStaging", Value: 0}))
				})
			})

			Context("when a desired app has crashed instances on some of the indices", func() {
				BeforeEach(func() {
					store.SyncDesiredState(a.DesiredState(3))

					store.SyncHeartbeats(dea.HeartbeatWith(
						a.InstanceAtIndex(0).Heartbeat(),
						a.InstanceAtIndex(1).Heartbeat(),
						a.InstanceAtIndex(2).Heartbeat(),

						a.CrashedInstanceHeartbeatAtIndex(1),
						a.CrashedInstanceHeartbeatAtIndex(2),
						a.CrashedInstanceHeartbeatAtIndex(2),
					))
				})
				It("should have the correct stats", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredApps", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredAppsPendingStaging", Value: 0}))
				})
			})

			Context("when a desired app has extra instances heartbeating", func() {
				BeforeEach(func() {
					store.SyncDesiredState(a.DesiredState(3))

					store.SyncHeartbeats(dea.HeartbeatWith(
						a.InstanceAtIndex(0).Heartbeat(),
						a.InstanceAtIndex(1).Heartbeat(),
						a.InstanceAtIndex(2).Heartbeat(),
						a.InstanceAtIndex(4).Heartbeat(),
						a.InstanceAtIndex(5).Heartbeat(),
						a.CrashedInstanceHeartbeatAtIndex(3),
					))
				})
				It("should have the correct stats", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 5}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredApps", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredAppsPendingStaging", Value: 0}))
				})
			})

			Context("when a desired app has at least one of the desired instances reporting as crashed", func() {
				BeforeEach(func() {
					store.SyncDesiredState(a.DesiredState(3))

					store.SyncHeartbeats(dea.HeartbeatWith(
						a.InstanceAtIndex(0).Heartbeat(),
						a.CrashedInstanceHeartbeatAtIndex(1),
						a.CrashedInstanceHeartbeatAtIndex(1),
						a.InstanceAtIndex(2).Heartbeat(),
					))
				})

				It("should have the correct stats", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 2}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 2}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredApps", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredAppsPendingStaging", Value: 0}))
				})
			})

			Context("when a desired app has at least one of the desired instances missing", func() {
				BeforeEach(func() {
					store.SyncDesiredState(a.DesiredState(3))

					store.SyncHeartbeats(dea.HeartbeatWith(
						a.InstanceAtIndex(0).Heartbeat(),
						a.InstanceAtIndex(2).Heartbeat(),
					))
				})

				It("should have the correct stats", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 2}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredApps", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredAppsPendingStaging", Value: 0}))
				})
			})

			Context("when a desired app has all of the desired instances missing", func() {
				BeforeEach(func() {
					store.SyncDesiredState(a.DesiredState(3))
				})

				It("should have the correct stats", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredApps", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredAppsPendingStaging", Value: 0}))
				})
			})

			Context("when there is an undesired app that is reporting as running", func() {
				BeforeEach(func() {
					b := dea.GetApp(1)
					store.SyncHeartbeats(dea.HeartbeatWith(
						a.InstanceAtIndex(0).Heartbeat(),
						a.CrashedInstanceHeartbeatAtIndex(1),
						a.InstanceAtIndex(2).Heartbeat(),
						b.InstanceAtIndex(0).Heartbeat(),
					))
				})

				It("should have the correct stats", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 2}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredAppsPendingStaging", Value: 0}))
				})
			})

			Context("when there is an undesired app that is reporting as crashed", func() {
				BeforeEach(func() {
					store.SyncHeartbeats(dea.HeartbeatWith(
						a.CrashedInstanceHeartbeatAtIndex(0),
						a.CrashedInstanceHeartbeatAtIndex(1),
					))
				})

				It("should have the correct stats", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 2}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 2}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfDesiredAppsPendingStaging", Value: 0}))
				})
			})
		})
	})
	It("should tell its health", func() {
		Ω(metricsServer.Ok()).Should(BeTrue())
	})
})
