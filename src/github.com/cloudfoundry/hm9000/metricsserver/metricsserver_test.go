package metricsserver_test

import (
	"github.com/cloudfoundry/hm9000/config"
	. "github.com/cloudfoundry/hm9000/metricsserver"
	"github.com/cloudfoundry/hm9000/models"
	storepackage "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	"github.com/cloudfoundry/hm9000/testhelpers/fakestoreadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/faketimeprovider"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Metrics Server", func() {
	var (
		store         storepackage.Store
		storeAdapter  *fakestoreadapter.FakeStoreAdapter
		timeProvider  *faketimeprovider.FakeTimeProvider
		metricsServer *MetricsServer
	)

	BeforeEach(func() {
		conf, _ := config.DefaultConfig()
		storeAdapter = fakestoreadapter.New()
		store = storepackage.NewStore(conf, storeAdapter, fakelogger.NewFakeLogger())
		timeProvider = &faketimeprovider.FakeTimeProvider{TimeToProvide: time.Unix(100, 0)}

		metricsServer = New(nil, nil, fakelogger.NewFakeLogger(), store, timeProvider, conf)
	})

	Describe("the returned context", func() {
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
			})
		})

		Context("when the store is fresh", func() {
			var a appfixture.AppFixture

			BeforeEach(func() {
				a = appfixture.NewAppFixture()
				store.BumpDesiredFreshness(time.Unix(0, 0))
				store.BumpActualFreshness(time.Unix(0, 0))
			})

			Context("when a desired app has all instances running", func() {
				BeforeEach(func() {
					store.SaveDesiredState(a.DesiredState(3))

					store.SaveActualState(
						a.InstanceAtIndex(0).Heartbeat(),
						a.InstanceAtIndex(1).Heartbeat(),
						a.InstanceAtIndex(2).Heartbeat(),
					)
				})

				It("should report the app as 100 %% reporting", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 0}))
				})
			})

			Context("when a desired app has an instance starting and others running", func() {
				BeforeEach(func() {
					store.SaveDesiredState(a.DesiredState(3))

					startingHB := a.InstanceAtIndex(1).Heartbeat()
					startingHB.State = models.InstanceStateStarting
					store.SaveActualState(
						a.InstanceAtIndex(0).Heartbeat(),
						startingHB,
						a.InstanceAtIndex(2).Heartbeat(),
					)
				})

				It("should report the app as 100 %% reporting", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 0}))
				})
			})

			Context("when a desired app has crashed instances on some of the indices", func() {
				BeforeEach(func() {
					store.SaveDesiredState(a.DesiredState(3))

					store.SaveActualState(
						a.InstanceAtIndex(0).Heartbeat(),
						a.InstanceAtIndex(1).Heartbeat(),
						a.InstanceAtIndex(2).Heartbeat(),

						a.CrashedInstanceHeartbeatAtIndex(1),
						a.CrashedInstanceHeartbeatAtIndex(2),
						a.CrashedInstanceHeartbeatAtIndex(2),
					)
				})
				It("should report the app as 100 %% reporting", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 0}))
				})
			})

			Context("when a desired app has extra instances heartbeating", func() {
				BeforeEach(func() {
					store.SaveDesiredState(a.DesiredState(3))

					store.SaveActualState(
						a.InstanceAtIndex(0).Heartbeat(),
						a.InstanceAtIndex(1).Heartbeat(),
						a.InstanceAtIndex(2).Heartbeat(),
						a.InstanceAtIndex(4).Heartbeat(),
						a.InstanceAtIndex(5).Heartbeat(),
						a.CrashedInstanceHeartbeatAtIndex(3),
					)
				})
				It("should report the app as 100 %% reporting", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 5}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 1}))
				})
			})

			Context("when a desired app has at least one of the desired instances reporting as crashed", func() {
				BeforeEach(func() {
					store.SaveDesiredState(a.DesiredState(3))

					store.SaveActualState(
						a.InstanceAtIndex(0).Heartbeat(),
						a.CrashedInstanceHeartbeatAtIndex(1),
						a.CrashedInstanceHeartbeatAtIndex(1),
						a.InstanceAtIndex(2).Heartbeat(),
					)
				})

				It("should report the app as 100 %% reporting", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 2}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 2}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 1}))
				})
			})

			Context("when a desired app has at least one of the desired instances missing", func() {
				BeforeEach(func() {
					store.SaveDesiredState(a.DesiredState(3))

					store.SaveActualState(
						a.InstanceAtIndex(0).Heartbeat(),
						a.InstanceAtIndex(2).Heartbeat(),
					)
				})

				It("should not report the app as 100 %% reporting", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 2}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 0}))
				})
			})

			Context("when a desired app has all of the desired instances missing", func() {
				BeforeEach(func() {
					store.SaveDesiredState(a.DesiredState(3))
				})

				It("should not report the app as 100 %% reporting", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 0}))
				})
			})

			Context("when there is an undesired app that is reporting as running", func() {
				BeforeEach(func() {
					b := appfixture.NewAppFixture()
					store.SaveActualState(
						a.InstanceAtIndex(0).Heartbeat(),
						a.CrashedInstanceHeartbeatAtIndex(1),
						a.InstanceAtIndex(2).Heartbeat(),
						b.InstanceAtIndex(0).Heartbeat(),
					)
				})

				It("should report the app as an undesired app", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 2}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 3}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 1}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 1}))
				})
			})

			Context("when there is an undesired app that is reporting as crashed", func() {
				BeforeEach(func() {
					store.SaveActualState(
						a.CrashedInstanceHeartbeatAtIndex(0),
						a.CrashedInstanceHeartbeatAtIndex(1),
					)
				})

				It("should report the app as an undesired app", func() {
					context := metricsServer.Emit()
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithAllInstancesReporting", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfAppsWithMissingInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfUndesiredRunningApps", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfRunningInstances", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfMissingIndices", Value: 0}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedInstances", Value: 2}))
					Ω(context.Metrics).Should(ContainElement(instrumentation.Metric{Name: "NumberOfCrashedIndices", Value: 2}))
				})
			})
		})
	})
	It("should tell its health", func() {
		Ω(metricsServer.Ok()).Should(BeTrue())
	})
})
