package md_test

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
		validRequest string
	)

	BeforeEach(func() {
		a = appfixture.NewAppFixture()
		validRequest = fmt.Sprintf(`{"droplet":"%s","version":"%s"}`, a.AppGuid, a.AppVersion)

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
			_, err := natsRunner.MessageBus.Subscribe(replyTo, func(message *yagnats.Message) {
				Ω(message.Payload).Should(ContainSubstring(`"droplet":"%s"`, a.AppGuid))
				Ω(message.Payload).Should(ContainSubstring(`"instances":2`))
				Ω(message.Payload).Should(ContainSubstring(`"instance":"%s"`, a.InstanceAtIndex(0).InstanceGuid))

				close(done)
			})
			Ω(err).ShouldNot(HaveOccured())

			err = natsRunner.MessageBus.PublishWithReplyTo("app.state", validRequest, replyTo)
			Ω(err).ShouldNot(HaveOccured())
		})
	})

	Context("when the store is not fresh", func() {
		BeforeEach(func() {
			simulator.Tick(simulator.TicksToAttainFreshness - 1)
			cliRunner.StartAPIServer(simulator.currentTimestamp)
		})

		It("should return -1 for all metrics", func(done Done) {
			replyTo := models.Guid()
			_, err := natsRunner.MessageBus.Subscribe(replyTo, func(message *yagnats.Message) {
				Ω(message.Payload).Should(Equal(`{}`))

				close(done)
			})
			Ω(err).ShouldNot(HaveOccured())

			err = natsRunner.MessageBus.PublishWithReplyTo("app.state", validRequest, replyTo)
			Ω(err).ShouldNot(HaveOccured())
		})
	})
})
