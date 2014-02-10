package evacuator_test

import (
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/models"
	storepackage "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	. "github.com/cloudfoundry/hm9000/testhelpers/custommatchers"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	"github.com/cloudfoundry/storeadapter/fakestoreadapter"
	"github.com/cloudfoundry/yagnats"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	"time"

	. "github.com/cloudfoundry/hm9000/evacuator"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Evacuator", func() {
	var (
		evacuator    *Evacuator
		messageBus   *fakeyagnats.FakeYagnats
		storeAdapter *fakestoreadapter.FakeStoreAdapter
		timeProvider *faketimeprovider.FakeTimeProvider

		store storepackage.Store
		app   appfixture.AppFixture
	)

	conf, _ := config.DefaultConfig()

	BeforeEach(func() {
		storeAdapter = fakestoreadapter.New()
		messageBus = fakeyagnats.New()
		store = storepackage.NewStore(conf, storeAdapter, fakelogger.NewFakeLogger())
		timeProvider = &faketimeprovider.FakeTimeProvider{
			TimeToProvide: time.Unix(100, 0),
		}

		app = appfixture.NewAppFixture()

		evacuator = New(messageBus, store, timeProvider, conf, fakelogger.NewFakeLogger())
		evacuator.Listen()
	})

	It("should be listening on the message bus for droplet.exited", func() {
		Ω(messageBus.Subscriptions).Should(HaveKey("droplet.exited"))
	})

	Context("when droplet.exited is received", func() {
		Context("when the message is malformed", func() {
			It("does nothing", func() {
				messageBus.Subscriptions["droplet.exited"][0].Callback(&yagnats.Message{
					Payload: []byte("ß"),
				})

				pendingStarts, err := store.GetPendingStartMessages()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(pendingStarts).Should(BeEmpty())
			})
		})

		Context("when the reason is DEA_EVACUATION", func() {
			BeforeEach(func() {
				messageBus.Subscriptions["droplet.exited"][0].Callback(&yagnats.Message{
					Payload: app.InstanceAtIndex(1).DropletExited(models.DropletExitedReasonDEAEvacuation).ToJSON(),
				})
			})

			It("should put a high priority pending start message (configured to skip verification) into the queue", func() {
				pendingStarts, err := store.GetPendingStartMessages()
				Ω(err).ShouldNot(HaveOccurred())

				expectedStartMessage := models.NewPendingStartMessage(timeProvider.Time(), 0, conf.GracePeriod(), app.AppGuid, app.AppVersion, 1, 2.0, models.PendingStartMessageReasonEvacuating)
				expectedStartMessage.SkipVerification = true

				Ω(pendingStarts).Should(ContainElement(EqualPendingStartMessage(expectedStartMessage)))
			})
		})

		Context("when the reason is DEA_SHUTDOWN", func() {
			BeforeEach(func() {
				messageBus.Subscriptions["droplet.exited"][0].Callback(&yagnats.Message{
					Payload: app.InstanceAtIndex(1).DropletExited(models.DropletExitedReasonDEAShutdown).ToJSON(),
				})
			})

			It("should put a high priority pending start message (configured to skip verification) into the queue", func() {
				pendingStarts, err := store.GetPendingStartMessages()
				Ω(err).ShouldNot(HaveOccurred())

				expectedStartMessage := models.NewPendingStartMessage(timeProvider.Time(), 0, conf.GracePeriod(), app.AppGuid, app.AppVersion, 1, 2.0, models.PendingStartMessageReasonEvacuating)
				expectedStartMessage.SkipVerification = true

				Ω(pendingStarts).Should(ContainElement(EqualPendingStartMessage(expectedStartMessage)))
			})
		})

		Context("when the reason is STOPPED", func() {
			BeforeEach(func() {
				messageBus.Subscriptions["droplet.exited"][0].Callback(&yagnats.Message{
					Payload: app.InstanceAtIndex(1).DropletExited(models.DropletExitedReasonStopped).ToJSON(),
				})
			})

			It("should do nothing", func() {
				pendingStarts, err := store.GetPendingStartMessages()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(pendingStarts).Should(BeEmpty())
			})
		})

		Context("when the reason is CRASHED", func() {
			BeforeEach(func() {
				messageBus.Subscriptions["droplet.exited"][0].Callback(&yagnats.Message{
					Payload: app.InstanceAtIndex(1).DropletExited(models.DropletExitedReasonCrashed).ToJSON(),
				})
			})

			It("should do nothing", func() {
				pendingStarts, err := store.GetPendingStartMessages()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(pendingStarts).Should(BeEmpty())
			})
		})
	})
})
