package apiserver_test

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
	"github.com/cloudfoundry/hm9000/apiserver"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/models"
	storepackage "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	"github.com/cloudfoundry/storeadapter/fakestoreadapter"
	"github.com/cloudfoundry/yagnats"
	"github.com/cloudfoundry/yagnats/fakeyagnats"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type AppResponse struct {
	AppGuid    string `json:"droplet"`
	AppVersion string `json:"version"`

	Desired            models.DesiredAppState     `json:"desired"`
	InstanceHeartbeats []models.InstanceHeartbeat `json:"instance_heartbeats"`
	CrashCounts        []models.CrashCount        `json:"crash_counts"`
}

type BulkAppResponse map[string]AppResponse

var _ = Describe("Apiserver", func() {
	var store storepackage.Store
	var storeAdapter *fakestoreadapter.FakeStoreAdapter
	var timeProvider *faketimeprovider.FakeTimeProvider
	var messageBus *fakeyagnats.FakeYagnats

	conf, _ := config.DefaultConfig()

	makeRequest := func(request string) (response AppResponse) {
		replyToGuid := models.Guid()
		messageBus.Subscriptions["app.state"][0].Callback(&yagnats.Message{
			Payload: []byte(request),
			ReplyTo: replyToGuid,
		})

		Ω(messageBus.PublishedMessages[replyToGuid]).Should(HaveLen(1))

		err := json.Unmarshal(messageBus.PublishedMessages[replyToGuid][0].Payload, &response)
		Ω(err).ShouldNot(HaveOccurred())

		return response
	}

	makeBulkRequest := func(request string) (response BulkAppResponse) {
		replyToGuid := models.Guid()
		messageBus.Subscriptions["app.state.bulk"][0].Callback(&yagnats.Message{
			Payload: []byte(request),
			ReplyTo: replyToGuid,
		})

		Ω(messageBus.PublishedMessages[replyToGuid]).Should(HaveLen(1))

		err := json.Unmarshal(messageBus.PublishedMessages[replyToGuid][0].Payload, &response)
		Ω(err).ShouldNot(HaveOccurred())

		return response
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
				response := makeRequest("{}")
				Ω(response).Should(BeZero())
			})
		})

		Context("when the request payload is invalid JSON", func() {
			It("responds with an empty hash", func() {
				response := makeRequest("ß")
				Ω(response).Should(BeZero())
			})
		})

		Context("when no reply-to is given", func() {
			It("should drop the request on the floor", func() {
				messageBus.Subscriptions["app.state"][0].Callback(&yagnats.Message{
					Payload: []byte("{}"),
				})

				Ω(messageBus.PublishedMessages).Should(BeEmpty())
			})
		})

		Context("when the request contains the droplet and version", func() {
			var app appfixture.AppFixture
			var expectedApp AppResponse
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
				expectedApp = AppResponse{
					AppGuid:            app.AppGuid,
					AppVersion:         app.AppVersion,
					Desired:            app.DesiredState(3),
					InstanceHeartbeats: instanceHeartbeats,
					CrashCounts:        []models.CrashCount{crashCount},
				}

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
						Ω(response).Should(BeZero())
					})
				})

				Context("when the app query parameters correspond to an existing app", func() {
					It("should return the actual instances and crashes of the app", func() {
						response := makeRequest(validRequestPayload)
						Ω(response.AppGuid).Should(Equal(expectedApp.AppGuid))
						Ω(response.AppVersion).Should(Equal(expectedApp.AppVersion))
						Ω(response.Desired).Should(Equal(expectedApp.Desired))
						Ω(response.InstanceHeartbeats).Should(ConsistOf(expectedApp.InstanceHeartbeats))
						Ω(response.CrashCounts).Should(ConsistOf(expectedApp.CrashCounts))
					})
				})

				Context("when something else goes wrong with the store", func() {
					BeforeEach(func() {
						storeAdapter.GetErrInjector = fakestoreadapter.NewFakeStoreAdapterErrorInjector("desired", fmt.Errorf("No desired state for you!"))
					})

					It("should return an empty hash", func() {
						response := makeRequest(validRequestPayload)
						Ω(response).Should(BeZero())
					})
				})
			})

			Context("when the store is not fresh", func() {
				It("should return an empty hash", func() {
					response := makeRequest(validRequestPayload)
					Ω(response).Should(BeZero())
				})
			})
		})
	})

	Context("responding to app.state.bulk", func() {
		Context("when the request is empty", func() {
			It("should return an empty hash", func() {
				response := makeBulkRequest("[]")
				Ω(response).Should(BeEmpty())
			})
		})

		Context("when the request payload is invalid JSON", func() {
			It("should return an empty hash", func() {
				response := makeBulkRequest("[]")
				Ω(response).Should(BeEmpty())
			})
		})

		Context("when no reply-to is given", func() {
			It("should drop the request on the floor", func() {
				messageBus.Subscriptions["app.state.bulk"][0].Callback(&yagnats.Message{
					Payload: []byte("{}"),
				})

				Ω(messageBus.PublishedMessages).Should(BeEmpty())
			})
		})

		Context("when the request contains a valid droplet and version", func() {
			var app appfixture.AppFixture
			var expectedApp AppResponse
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
				expectedApp = AppResponse{
					AppGuid:            app.AppGuid,
					AppVersion:         app.AppVersion,
					Desired:            app.DesiredState(3),
					InstanceHeartbeats: instanceHeartbeats,
					CrashCounts:        []models.CrashCount{crashCount},
				}

				store.SyncDesiredState(app.DesiredState(3))
				store.SyncHeartbeats(app.Heartbeat(3))
				store.SaveCrashCounts(crashCount)
				validRequestPayload = fmt.Sprintf(`[{"droplet":"%s","version":"%s"}]`, app.AppGuid, app.AppVersion)
			})

			Context("when the store is fresh", func() {
				BeforeEach(func() {
					store.BumpDesiredFreshness(time.Unix(0, 0))
					store.BumpActualFreshness(time.Unix(0, 0))
				})

				Context("when the app query parameters do not correspond to an existing app", func() {
					It("should respond with an empty hash", func() {
						response := makeBulkRequest(`[{"droplet":"elephant","version":"pink-flamingo"}]`)
						Ω(response).Should(BeEmpty())
					})
				})

				Context("when the app query parameters correspond to an existing app", func() {
					It("should return the actual instances and crashes of the app", func() {
						response := makeBulkRequest(validRequestPayload)
						Ω(response).Should(HaveLen(1))
						Ω(response).Should(HaveKey(expectedApp.AppGuid))
						receivedApp := response[expectedApp.AppGuid]
						Ω(receivedApp.AppGuid).Should(Equal(expectedApp.AppGuid))
						Ω(receivedApp.AppVersion).Should(Equal(expectedApp.AppVersion))
						Ω(receivedApp.Desired).Should(Equal(expectedApp.Desired))
						Ω(receivedApp.InstanceHeartbeats).Should(ConsistOf(expectedApp.InstanceHeartbeats))
						Ω(receivedApp.CrashCounts).Should(ConsistOf(expectedApp.CrashCounts))
					})
				})

				Context("when some of the apps are not found", func() {
					It("responds with the apps that are present", func() {
						validRequestPayload = fmt.Sprintf(`[{"droplet":"%s","version":"%s"},{"droplet":"jam-sandwich","version":"123"}]`, app.AppGuid, app.AppVersion)
						response := makeBulkRequest(validRequestPayload)
						Ω(response).Should(HaveLen(1))
						Ω(response).Should(HaveKey(expectedApp.AppGuid))
						receivedApp := response[expectedApp.AppGuid]
						Ω(receivedApp.AppGuid).Should(Equal(expectedApp.AppGuid))
						Ω(receivedApp.AppVersion).Should(Equal(expectedApp.AppVersion))
						Ω(receivedApp.Desired).Should(Equal(expectedApp.Desired))
						Ω(receivedApp.InstanceHeartbeats).Should(ConsistOf(expectedApp.InstanceHeartbeats))
						Ω(receivedApp.CrashCounts).Should(ConsistOf(expectedApp.CrashCounts))
					})
				})
			})

			Context("when the store is not fresh", func() {
				It("should return an empty hash", func() {
					response := makeBulkRequest(validRequestPayload)
					Ω(response).Should(BeEmpty())
				})
			})
		})
	})

})
