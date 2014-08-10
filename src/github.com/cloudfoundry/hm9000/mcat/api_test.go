package mcat_test

import (
	"fmt"

	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	"github.com/cloudfoundry/yagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Serving API", func() {
	var (
		a            appfixture.AppFixture
		validRequest []byte
	)

	Describe("app.state", func() {
		BeforeEach(func() {
			a = appfixture.NewAppFixture()
			validRequest = []byte(fmt.Sprintf(`{"droplet":"%s","version":"%s"}`, a.AppGuid, a.AppVersion))

			simulator.SetDesiredState(a.DesiredState(2))
			simulator.SetCurrentHeartbeats(a.Heartbeat(1))
		})

		AfterEach(func() {
			cliRunner.StopAPIServer()
		})

		Context("when the store is fresh", func() {
			BeforeEach(func() {
				simulator.Tick(simulator.TicksToAttainFreshness)
				cliRunner.StartAPIServer(simulator.currentTimestamp)
			})

			It("should return the app", func(done Done) {
				replyTo := models.Guid()
				_, err := coordinator.MessageBus.Subscribe(replyTo, func(message *yagnats.Message) {
					defer GinkgoRecover()
					Ω(string(message.Payload)).Should(ContainSubstring(`"droplet":"%s"`, a.AppGuid))
					Ω(string(message.Payload)).Should(ContainSubstring(`"instances":2`))
					Ω(string(message.Payload)).Should(ContainSubstring(`"instance":"%s"`, a.InstanceAtIndex(0).InstanceGuid))

					close(done)
				})
				Ω(err).ShouldNot(HaveOccurred())

				err = coordinator.MessageBus.PublishWithReplyTo("app.state", replyTo, validRequest)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Context("when the store is not fresh", func() {
			BeforeEach(func() {
				simulator.Tick(simulator.TicksToAttainFreshness - 1)
				cliRunner.StartAPIServer(simulator.currentTimestamp)
			})

			It("should return -1 for all metrics", func(done Done) {
				replyTo := models.Guid()
				_, err := coordinator.MessageBus.Subscribe(replyTo, func(message *yagnats.Message) {
					defer GinkgoRecover()
					Ω(string(message.Payload)).Should(BeEquivalentTo(`{}`))

					close(done)
				})
				Ω(err).ShouldNot(HaveOccurred())

				err = coordinator.MessageBus.PublishWithReplyTo("app.state", replyTo, validRequest)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
	})

	Describe("app.state.bulk", func() {
		var b appfixture.AppFixture

		BeforeEach(func() {
			a = appfixture.NewAppFixture()
			b = appfixture.NewAppFixture()
			validRequest = []byte(fmt.Sprintf(`[{"droplet":"%s","version":"%s"}, {"droplet":"%s","version":"%s"}]`, a.AppGuid, a.AppVersion, b.AppGuid, b.AppVersion))

			simulator.SetDesiredState(a.DesiredState(2), b.DesiredState(3))
			simulator.SetCurrentHeartbeats(a.Heartbeat(1), b.Heartbeat(1))
		})

		AfterEach(func() {
			cliRunner.StopAPIServer()
		})

		Context("when the store is fresh", func() {
			BeforeEach(func() {
				simulator.Tick(simulator.TicksToAttainFreshness)
				cliRunner.StartAPIServer(simulator.currentTimestamp)
			})

			It("should return the apps", func(done Done) {
				replyTo := models.Guid()
				_, err := coordinator.MessageBus.Subscribe(replyTo, func(message *yagnats.Message) {
					defer GinkgoRecover()
					Ω(string(message.Payload)).Should(ContainSubstring(`"droplet":"%s"`, a.AppGuid))
					Ω(string(message.Payload)).Should(ContainSubstring(`"droplet":"%s"`, b.AppGuid))
					Ω(string(message.Payload)).Should(ContainSubstring(`"instances":2`))
					Ω(string(message.Payload)).Should(ContainSubstring(`"instances":3`))

					close(done)
				})
				Ω(err).ShouldNot(HaveOccurred())

				err = coordinator.MessageBus.PublishWithReplyTo("app.state.bulk", replyTo, validRequest)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})

		Context("when the store is not fresh", func() {
			BeforeEach(func() {
				simulator.Tick(simulator.TicksToAttainFreshness - 1)
				cliRunner.StartAPIServer(simulator.currentTimestamp)
			})

			It("should return -1 for all metrics", func(done Done) {
				replyTo := models.Guid()
				_, err := coordinator.MessageBus.Subscribe(replyTo, func(message *yagnats.Message) {
					defer GinkgoRecover()
					Ω(string(message.Payload)).Should(BeEquivalentTo(`{}`))

					close(done)
				})
				Ω(err).ShouldNot(HaveOccurred())

				err = coordinator.MessageBus.PublishWithReplyTo("app.state.bulk", replyTo, validRequest)
				Ω(err).ShouldNot(HaveOccurred())
			})
		})
	})
})
