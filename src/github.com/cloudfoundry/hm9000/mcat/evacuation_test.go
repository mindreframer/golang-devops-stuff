package mcat_test

import (
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Evacuation and Shutdown", func() {
	var dea appfixture.DeaFixture
	var app appfixture.AppFixture

	BeforeEach(func() {
		dea = appfixture.NewDeaFixture()
		app = dea.GetApp(0)
		simulator.SetCurrentHeartbeats(dea.HeartbeatWith(app.InstanceAtIndex(0).Heartbeat()))
		simulator.SetDesiredState(app.DesiredState(1))
		simulator.Tick(simulator.TicksToAttainFreshness)
	})

	Describe("Shutdown handling by the evacuator component", func() {
		Context("when a SHUTDOWN droplet.exited message comes in", func() {
			BeforeEach(func() {
				cliRunner.StartEvacuator(simulator.currentTimestamp)
				coordinator.MessageBus.Publish("droplet.exited", app.InstanceAtIndex(0).DropletExited(models.DropletExitedReasonDEAShutdown).ToJSON())
			})

			AfterEach(func() {
				cliRunner.StopEvacuator()
			})

			It("should immediately start the app", func() {
				simulator.Tick(1)
				Ω(startStopListener.Starts).Should(HaveLen(1))
				Ω(startStopListener.Starts[0].AppGuid).Should(Equal(app.AppGuid))
				Ω(startStopListener.Starts[0].AppVersion).Should(Equal(app.AppVersion))
				Ω(startStopListener.Starts[0].InstanceIndex).Should(Equal(0))
			})
		})
	})

	Describe("Deterministic evacuation", func() {
		Context("when an app enters the evacuation state", func() {
			var evacuatingHeartbeat models.InstanceHeartbeat

			BeforeEach(func() {
				Ω(startStopListener.Starts).Should(BeEmpty())
				Ω(startStopListener.Stops).Should(BeEmpty())
				evacuatingHeartbeat = app.InstanceAtIndex(0).Heartbeat()
				evacuatingHeartbeat.State = models.InstanceStateEvacuating

				simulator.SetCurrentHeartbeats(dea.HeartbeatWith(evacuatingHeartbeat))
				simulator.Tick(1)
			})

			It("should immediately start the app", func() {
				Ω(startStopListener.Starts).Should(HaveLen(1))
				Ω(startStopListener.Starts[0].AppGuid).Should(Equal(app.AppGuid))
				Ω(startStopListener.Starts[0].AppVersion).Should(Equal(app.AppVersion))
				Ω(startStopListener.Starts[0].InstanceIndex).Should(Equal(0))
				Ω(startStopListener.Stops).Should(BeEmpty())
			})

			Context("when the app starts", func() {
				BeforeEach(func() {
					startStopListener.Reset()
					runningHeartbeat := app.InstanceAtIndex(0).Heartbeat()
					runningHeartbeat.InstanceGuid = models.Guid()
					simulator.SetCurrentHeartbeats(dea.HeartbeatWith(evacuatingHeartbeat))
					simulator.SetCurrentHeartbeats(models.Heartbeat{
						DeaGuid:            "new-dea",
						InstanceHeartbeats: []models.InstanceHeartbeat{runningHeartbeat},
					})
					simulator.Tick(1)
				})

				It("should stop the evacuated instance", func() {
					Ω(startStopListener.Starts).Should(BeEmpty())
					Ω(startStopListener.Stops).Should(HaveLen(1))
					Ω(startStopListener.Stops[0].AppGuid).Should(Equal(app.AppGuid))
					Ω(startStopListener.Stops[0].AppVersion).Should(Equal(app.AppVersion))
					Ω(startStopListener.Stops[0].InstanceGuid).Should(Equal(evacuatingHeartbeat.InstanceGuid))
				})
			})
		})
	})
})
