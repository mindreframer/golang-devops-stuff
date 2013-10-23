package md_test

import (
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Crashes", func() {
	var (
		a                 appfixture.AppFixture
		crashingHeartbeat models.Heartbeat
	)

	BeforeEach(func() {
		a = appfixture.NewAppFixture()
	})

	Describe("when all instances are crashed", func() {
		BeforeEach(func() {
			simulator.SetDesiredState(a.DesiredState(3))

			crashingHeartbeat = models.Heartbeat{
				DeaGuid: models.Guid(),
				InstanceHeartbeats: []models.InstanceHeartbeat{
					a.CrashedInstanceHeartbeatAtIndex(0),
					a.CrashedInstanceHeartbeatAtIndex(1),
					a.CrashedInstanceHeartbeatAtIndex(2),
				},
			}

			simulator.SetCurrentHeartbeats(crashingHeartbeat)
			simulator.Tick(simulator.TicksToAttainFreshness)
		})

		It("should only try to start instance at index 0", func() {
			Ω(startStopListener.Starts).Should(HaveLen(1))
			Ω(startStopListener.Starts[0].AppVersion).Should(Equal(a.AppVersion))
			Ω(startStopListener.Starts[0].InstanceIndex).Should(Equal(0))
		})

		It("should never try to stop crashes", func() {
			Ω(startStopListener.Stops).Should(BeEmpty())
			simulator.Tick(1)
			Ω(startStopListener.Stops).Should(BeEmpty())
		})
	})

	Describe("when at least one instance is running", func() {
		BeforeEach(func() {
			simulator.SetDesiredState(a.DesiredState(3))

			crashingHeartbeat = models.Heartbeat{
				DeaGuid: models.Guid(),
				InstanceHeartbeats: []models.InstanceHeartbeat{
					a.CrashedInstanceHeartbeatAtIndex(0),
					a.InstanceAtIndex(1).Heartbeat(),
					a.CrashedInstanceHeartbeatAtIndex(2),
				},
			}

			simulator.SetCurrentHeartbeats(crashingHeartbeat)
			simulator.Tick(simulator.TicksToAttainFreshness)
		})

		It("should start all the crashed instances", func() {
			Ω(startStopListener.Stops).Should(BeEmpty())
			Ω(startStopListener.Starts).Should(HaveLen(2))

			indicesToStart := []int{
				startStopListener.Starts[0].InstanceIndex,
				startStopListener.Starts[1].InstanceIndex,
			}

			Ω(indicesToStart).Should(ContainElement(0))
			Ω(indicesToStart).Should(ContainElement(2))
		})
	})

	Describe("the backoff policy", func() {
		BeforeEach(func() {
			simulator.SetDesiredState(a.DesiredState(2))

			crashingHeartbeat = models.Heartbeat{
				DeaGuid: models.Guid(),
				InstanceHeartbeats: []models.InstanceHeartbeat{
					a.InstanceAtIndex(0).Heartbeat(),
					a.CrashedInstanceHeartbeatAtIndex(1),
				},
			}

			simulator.SetCurrentHeartbeats(crashingHeartbeat)
			simulator.Tick(simulator.TicksToAttainFreshness)
		})

		Context("when the app keeps crashing", func() {
			It("should keep restarting the app instance with an appropriate backoff", func() {
				//crash #2
				simulator.Tick(simulator.GracePeriod)
				startStopListener.Reset()
				simulator.Tick(1)
				Ω(startStopListener.Starts).Should(HaveLen(1))

				//crash #3
				simulator.Tick(simulator.GracePeriod)
				startStopListener.Reset()
				simulator.Tick(1)
				Ω(startStopListener.Starts).Should(HaveLen(1))

				//crash #4, backoff begins
				simulator.Tick(simulator.GracePeriod)
				startStopListener.Reset()
				simulator.Tick(1)
				Ω(startStopListener.Starts).Should(HaveLen(0))

				//take more ticks longer to send a new start messages
				simulator.Tick(simulator.GracePeriod)
				Ω(startStopListener.Starts).Should(HaveLen(1))
			})
		})

		Context("when the app starts running", func() {
			BeforeEach(func() {
				//crash #2
				simulator.Tick(simulator.GracePeriod) //wait for keep-alive to expire
				simulator.Tick(1)                     //sends start for #2

				//crash #3
				simulator.Tick(simulator.GracePeriod) //wait for keep-alive #2 to expire
				simulator.Tick(1)                     //sends start for #3

				simulator.Tick(simulator.GracePeriod) //wait for keep-alive #3 to expire
				runningHeartbeat := models.Heartbeat{
					DeaGuid: models.Guid(),
					InstanceHeartbeats: []models.InstanceHeartbeat{
						a.InstanceAtIndex(0).Heartbeat(),
						a.InstanceAtIndex(1).Heartbeat(),
						a.CrashedInstanceHeartbeatAtIndex(1),
					},
				}

				startStopListener.Reset()
				simulator.SetCurrentHeartbeats(runningHeartbeat)
				simulator.Tick(1) //app is running, no starts should be scheduled
				Ω(startStopListener.Starts).Should(HaveLen(0))
			})

			Context("when it starts crashing again *before* the crash count expires", func() {
				It("should continue the backoff policy where it left off", func() {
					simulator.SetCurrentHeartbeats(crashingHeartbeat)
					simulator.Tick(simulator.TicksToExpireHeartbeat) //kill off the running heartbeat and then schedule a start
					Ω(startStopListener.Starts).Should(HaveLen(0))
					simulator.Tick(simulator.GracePeriod)
					Ω(startStopListener.Starts).Should(HaveLen(1))
				})
			})

			Context("when it starts crashing again *after* the crash count expires", func() {
				It("should reset the backoff policy", func() {
					simulator.Tick(6 * 2) //6 is the maximum backoff (cli_runner_test sets this in the config) and the crash count TTL is max backoff * 2
					simulator.SetCurrentHeartbeats(crashingHeartbeat)
					simulator.Tick(simulator.TicksToExpireHeartbeat) //kill off the running heartbeat and then schedule a start
					simulator.Tick(1)
					Ω(startStopListener.Starts).Should(HaveLen(1))
				})
			})
		})
	})
})
