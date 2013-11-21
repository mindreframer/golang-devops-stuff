package appfixture_test

import (
	. "github.com/cloudfoundry/hm9000/models"
	. "github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("App Fixture", func() {
	var app AppFixture

	BeforeEach(func() {
		app = NewAppFixture()
	})

	It("makes a reasonable app", func() {
		Ω(app.AppGuid).ShouldNot(BeEmpty())
		Ω(app.AppVersion).ShouldNot(BeEmpty())
	})

	Describe("the desired state message", func() {
		It("generates one with sane defaults", func() {
			desired := app.DesiredState(1)

			Ω(desired.AppGuid).Should(Equal(app.AppGuid))
			Ω(desired.AppVersion).Should(Equal(app.AppVersion))
			Ω(desired.NumberOfInstances).Should(BeNumerically("==", 1))
			Ω(desired.State).Should(Equal(AppStateStarted))
			Ω(desired.PackageState).Should(Equal(AppPackageStateStaged))
		})
	})

	Describe("InstanceAtIndex", func() {
		It("creates and memoizes instance", func() {
			instance := app.InstanceAtIndex(0)

			Ω(instance.AppGuid).Should(Equal(app.AppGuid))
			Ω(instance.AppVersion).Should(Equal(app.AppVersion))
			Ω(instance.InstanceGuid).ShouldNot(BeEmpty())
			Ω(instance.InstanceIndex).Should(Equal(0))

			instanceAgain := app.InstanceAtIndex(0)
			Ω(instanceAgain).Should(Equal(instance))

			otherInstance := app.InstanceAtIndex(3)
			Ω(otherInstance.InstanceIndex).Should(Equal(3))
			Ω(otherInstance.InstanceGuid).ShouldNot(Equal(instance.InstanceGuid))
		})
	})

	Describe("CrashedInstanceHeartbeatAtIndex", func() {
		It("should create an instance heartbeat, in the crashed state, at the passed in index", func() {
			index := 1
			heartbeat := app.CrashedInstanceHeartbeatAtIndex(index)
			Ω(heartbeat.State).Should(Equal(InstanceStateCrashed))
			Ω(heartbeat.AppGuid).Should(Equal(app.AppGuid))
			Ω(heartbeat.AppVersion).Should(Equal(app.AppVersion))
			Ω(heartbeat.InstanceGuid).ShouldNot(BeZero())
			Ω(heartbeat.InstanceIndex).Should(Equal(index))
			Ω(heartbeat.DeaGuid).Should(Equal(app.DeaGuid))
		})
	})

	Describe("Instance", func() {
		var instance Instance
		BeforeEach(func() {
			instance = app.InstanceAtIndex(0)
		})

		Describe("Heartbeat", func() {
			It("creates an instance heartbeat", func() {
				heartbeat := instance.Heartbeat()

				Ω(heartbeat.AppGuid).Should(Equal(app.AppGuid))
				Ω(heartbeat.AppVersion).Should(Equal(app.AppVersion))
				Ω(heartbeat.InstanceGuid).Should(Equal(instance.InstanceGuid))
				Ω(heartbeat.InstanceIndex).Should(Equal(instance.InstanceIndex))
				Ω(heartbeat.State).Should(Equal(InstanceStateRunning))
				Ω(heartbeat.DeaGuid).Should(Equal(app.DeaGuid))
			})
		})

		Describe("DropletExited", func() {
			It("returns droplet exited with the passed in reason", func() {
				exited := instance.DropletExited(DropletExitedReasonStopped)

				Ω(exited.CCPartition).Should(Equal("default"))
				Ω(exited.AppGuid).Should(Equal(app.AppGuid))
				Ω(exited.AppVersion).Should(Equal(app.AppVersion))
				Ω(exited.InstanceGuid).Should(Equal(instance.InstanceGuid))
				Ω(exited.InstanceIndex).Should(Equal(instance.InstanceIndex))
				Ω(exited.Reason).Should(Equal(DropletExitedReasonStopped))
				Ω(exited.ExitStatusCode).Should(Equal(0))
				Ω(exited.ExitDescription).Should(Equal("exited"))
				Ω(exited.CrashTimestamp).Should(BeZero())
			})
		})
	})

	Describe("Heartbeat", func() {
		It("creates a heartbeat for the desired number of instances, using the correct instnace guids when available", func() {
			instance := app.InstanceAtIndex(0)
			heartbeat := app.Heartbeat(2)

			Ω(heartbeat.DeaGuid).ShouldNot(BeEmpty())
			Ω(heartbeat.DeaGuid).Should(Equal(app.DeaGuid))

			Ω(heartbeat.InstanceHeartbeats).Should(HaveLen(2))
			Ω(heartbeat.InstanceHeartbeats[0]).Should(Equal(instance.Heartbeat()))
			Ω(heartbeat.InstanceHeartbeats[1]).Should(Equal(app.InstanceAtIndex(1).Heartbeat()))

			Ω(app.Heartbeat(2)).Should(Equal(heartbeat))
		})
	})

	Describe("Droplet Updated", func() {
		It("creates a droplet.updated message with the correct guid", func() {
			droplet_updated := app.DropletUpdated()

			Ω(droplet_updated.AppGuid).Should(Equal(app.AppGuid))
		})
	})
})
