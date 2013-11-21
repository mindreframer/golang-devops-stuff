package apiserver_test

import (
	"fmt"
	"github.com/cloudfoundry/hm9000/apiserver"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/models"
	storepackage "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	"github.com/cloudfoundry/hm9000/testhelpers/fakestoreadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/faketimeprovider"
	"github.com/cloudfoundry/yagnats"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Apiserver", func() {
	var store storepackage.Store
	var storeAdapter *fakestoreadapter.FakeStoreAdapter
	var timeProvider *faketimeprovider.FakeTimeProvider
	var messageBus *fakeyagnats.FakeYagnats

	conf, _ := config.DefaultConfig()

	makeRequest := func(request string) (response string) {
		replyToGuid := models.Guid()
		messageBus.Subscriptions["app.state"][0].Callback(&yagnats.Message{
			Payload: request,
			ReplyTo: replyToGuid,
		})

		Ω(messageBus.PublishedMessages[replyToGuid]).Should(HaveLen(1))
		return messageBus.PublishedMessages[replyToGuid][0].Payload
	}

	BeforeEach(func() {
		messageBus = fakeyagnats.New()
		storeAdapter = fakestoreadapter.New()
		store = storepackage.NewStore(conf, storeAdapter, fakelogger.NewFakeLogger())
		timeProvider = &faketimeprovider.FakeTimeProvider{
			TimeToProvide: time.Unix(100, 0),
		}

		server := apiserver.New(messageBus, store, timeProvider, fakelogger.NewFakeLogger())
		server.Listen()
	})

	It("should subscribe on a queue", func() {
		Ω(messageBus.Subscriptions["app.state"]).ShouldNot(BeEmpty())
		subscription := messageBus.Subscriptions["app.state"][0]
		Ω(subscription.Queue).Should(Equal("hm9000"))
	})

	Context("responding to app.state", func() {
		Context("when the request is empty", func() {
			It("should return an empty hash", func() {
				body := makeRequest("{}")
				Ω(body).Should(Equal("{}"))
			})
		})

		Context("when the request payload is invalid JSON", func() {
			It("responds with an empty hash", func() {
				body := makeRequest("ß")
				Ω(body).Should(Equal("{}"))
			})
		})

		Context("when no reply-to is given", func() {
			It("should drop the request on the floor", func() {
				messageBus.Subscriptions["app.state"][0].Callback(&yagnats.Message{
					Payload: "{}",
				})

				Ω(messageBus.PublishedMessages).Should(BeEmpty())
			})
		})

		Context("when the request contains the droplet and version", func() {
			var app appfixture.AppFixture
			var expectedApp *models.App
			var validRequestPayload string

			BeforeEach(func() {
				app = appfixture.NewAppFixture()
				instanceHeartbeats := []models.InstanceHeartbeat{
					app.InstanceAtIndex(0).Heartbeat(),
					app.InstanceAtIndex(1).Heartbeat(),
					app.InstanceAtIndex(2).Heartbeat(),
				}
				crashCount := models.CrashCount{
					AppGuid:       app.AppGuid,
					AppVersion:    app.AppVersion,
					InstanceIndex: 1,
					CrashCount:    2,
				}
				expectedApp = models.NewApp(
					app.AppGuid,
					app.AppVersion,
					app.DesiredState(3),
					instanceHeartbeats,
					map[int]models.CrashCount{1: crashCount},
				)

				store.SyncDesiredState(app.DesiredState(3))
				store.SyncHeartbeats(app.Heartbeat(3))
				store.SaveCrashCounts(crashCount)
				validRequestPayload = fmt.Sprintf(`{"droplet":"%s","version":"%s"}`, app.AppGuid, app.AppVersion)
			})

			Context("when the store is fresh", func() {
				BeforeEach(func() {
					store.BumpDesiredFreshness(time.Unix(0, 0))
					store.BumpActualFreshness(time.Unix(0, 0))
				})

				Context("when the app query parameters do not correspond to an existing app", func() {
					It("should respond with an empty hash", func() {
						response := makeRequest(`{"droplet":"elephant","version":"pink-flamingo"}`)
						Ω(response).Should(Equal("{}"))
					})
				})

				Context("when the app query parameters correspond to an existing app", func() {
					It("should return the actual instances and crashes of the app", func() {
						response := makeRequest(validRequestPayload)
						Ω(response).Should(Equal(string(expectedApp.ToJSON())))
					})
				})

				Context("when something else goes wrong with the store", func() {
					BeforeEach(func() {
						storeAdapter.GetErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector("desired", fmt.Errorf("No desired state for you!"))
					})

					It("should 500", func() {
						response := makeRequest(validRequestPayload)
						Ω(response).Should(Equal("{}"))
					})
				})
			})

			Context("when the store is not fresh", func() {
				It("should return a 404", func() {
					response := makeRequest(validRequestPayload)
					Ω(response).Should(Equal("{}"))
				})
			})
		})
	})
})
