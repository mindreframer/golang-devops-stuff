package mcat_test

import (
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Simple Cases Test", func() {
	var app1, app2 appfixture.AppFixture

	BeforeEach(func() {
		app1 = appfixture.NewAppFixture()
		app2 = appfixture.NewAppFixture()
	})

	Context("when all running instances are desired", func() {
		BeforeEach(func() {
			simulator.SetCurrentHeartbeats(app1.Heartbeat(1), app2.Heartbeat(1))
			simulator.SetDesiredState(app1.DesiredState(1), app2.DesiredState(1))
			simulator.Tick(simulator.TicksToAttainFreshness)
			simulator.Tick(1)
		})

		It("should not send any messages", func() {
			Ω(startStopListener.Starts).Should(BeEmpty())
			Ω(startStopListener.Stops).Should(BeEmpty())
		})
	})

	Context("when a desired app is pending staging", func() {
		Context("and it has a running instance", func() {
			BeforeEach(func() {
				desired := app1.DesiredState(1)
				desired.PackageState = models.AppPackageStatePending
				simulator.SetDesiredState(desired)
				simulator.SetCurrentHeartbeats(app1.Heartbeat(1))
				simulator.Tick(simulator.TicksToAttainFreshness)
				simulator.Tick(1)
			})

			It("should not try to stop that instance", func() {
				Ω(startStopListener.Starts).Should(BeEmpty())
				Ω(startStopListener.Stops).Should(BeEmpty())
			})
		})
	})

	Context("when there is a missing instance", func() {
		BeforeEach(func() {
			simulator.SetCurrentHeartbeats(app1.Heartbeat(1), app2.Heartbeat(1))
			simulator.SetDesiredState(app1.DesiredState(1), app2.DesiredState(2))
			simulator.Tick(simulator.TicksToAttainFreshness) //this tick will schedule a start

			// no message is sent during the start send message delay
			simulator.Tick(1)
			Ω(startStopListener.Starts).Should(BeEmpty())

			simulator.Tick(1)
			Ω(startStopListener.Starts).Should(BeEmpty())
		})

		Context("when the instance recovers on its own", func() {
			BeforeEach(func() {
				simulator.SetCurrentHeartbeats(app1.Heartbeat(1), app2.Heartbeat(2))
				simulator.Tick(1)
			})

			It("should not send a start message", func() {
				Ω(startStopListener.Starts).Should(HaveLen(0))
			})
		})

		Context("when the instance is no longer desired", func() {
			BeforeEach(func() {
				simulator.SetDesiredState(app1.DesiredState(1), app2.DesiredState(1))
				simulator.Tick(1)
			})

			It("should not send a start message", func() {
				Ω(startStopListener.Starts).Should(HaveLen(0))
			})
		})

		Context("when the instance does not recover on its own", func() {
			BeforeEach(func() {
				simulator.Tick(1)
			})

			It("should send a start message, after a delay, for the missing instance", func() {
				Ω(startStopListener.Starts).Should(HaveLen(1))

				start := startStopListener.Starts[0]
				Ω(start.AppGuid).Should(Equal(app2.AppGuid))
				Ω(start.AppVersion).Should(Equal(app2.AppVersion))
				Ω(start.InstanceIndex).Should(Equal(1))
			})
		})
	})

	Context("when there is an undesired instance running", func() {
		BeforeEach(func() {
			simulator.SetDesiredState(app2.DesiredState(1))
			simulator.SetCurrentHeartbeats(app2.Heartbeat(2))
			simulator.Tick(simulator.TicksToAttainFreshness)
		})

		Context("when the instance becomes desired", func() {
			BeforeEach(func() {
				simulator.SetDesiredState(app2.DesiredState(2))
				startStopListener.Reset()
				simulator.Tick(1)
			})

			It("should not send a stop message", func() {
				Ω(startStopListener.Stops).Should(HaveLen(0))
			})
		})

		Context("when the app is still running", func() {
			BeforeEach(func() {
				simulator.Tick(1)
			})

			It("should send a stop message, immediately, for the missing instance", func() {
				Ω(startStopListener.Stops).Should(HaveLen(1))

				stop := startStopListener.Stops[0]
				Ω(stop.AppGuid).Should(Equal(app2.AppGuid))
				Ω(stop.AppVersion).Should(Equal(app2.AppVersion))
				Ω(stop.InstanceGuid).Should(Equal(app2.InstanceAtIndex(1).InstanceGuid))
				Ω(stop.InstanceIndex).Should(Equal(1))
				Ω(stop.IsDuplicate).Should(BeFalse())
			})
		})
	})
})
