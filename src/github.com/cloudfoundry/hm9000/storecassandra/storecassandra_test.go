package storecassandra_test

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/models"
	storepackage "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/storeadapter"
	. "github.com/cloudfoundry/hm9000/storecassandra"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	. "github.com/cloudfoundry/hm9000/testhelpers/custommatchers"
	"github.com/cloudfoundry/hm9000/testhelpers/faketimeprovider"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
	"tux21b.org/v1/gocql"
)

var _ = Describe("Storecassandra", func() {
	var store *StoreCassandra
	var timeProvider *faketimeprovider.FakeTimeProvider
	var dea appfixture.DeaFixture
	var app1 appfixture.AppFixture
	var app2 appfixture.AppFixture

	var crashCount1 models.CrashCount
	var crashCount2 models.CrashCount

	var startMessage1 models.PendingStartMessage
	var startMessage2 models.PendingStartMessage

	var stopMessage1 models.PendingStopMessage
	var stopMessage2 models.PendingStopMessage

	conf, _ := config.DefaultConfig()

	BeforeEach(func() {
		timeProvider = &faketimeprovider.FakeTimeProvider{
			TimeToProvide: time.Unix(100, 0),
		}

		var err error
		store, err = New(cassandraRunner.NodeURLS(), gocql.One, conf, timeProvider)
		Ω(err).ShouldNot(HaveOccured())

		dea = appfixture.NewDeaFixture()
		app1 = dea.GetApp(0)
		app2 = dea.GetApp(1)
	})

	Describe("Syncing and reading Desired State", func() {
		BeforeEach(func() {
			err := store.SyncDesiredState(app1.DesiredState(1), app2.DesiredState(3))
			Ω(err).ShouldNot(HaveOccured())
		})

		It("should return the stored desired state", func() {
			state, err := store.GetDesiredState()
			Ω(err).ShouldNot(HaveOccured())
			Ω(state).Should(HaveLen(2))

			Ω(state[app1.DesiredState(1).StoreKey()]).Should(EqualDesiredState(app1.DesiredState(1)))
			Ω(state[app2.DesiredState(3).StoreKey()]).Should(EqualDesiredState(app2.DesiredState(3)))
		})

		Context("When resyncing the desired state", func() {
			var app3 appfixture.AppFixture

			BeforeEach(func() {
				app3 = dea.GetApp(2)

				err := store.SyncDesiredState(app2.DesiredState(2), app3.DesiredState(1))
				Ω(err).ShouldNot(HaveOccured())
			})

			It("should update any changed state, remove any stale state, and add any new state", func() {
				state, err := store.GetDesiredState()
				Ω(err).ShouldNot(HaveOccured())
				Ω(state).Should(HaveLen(2))

				Ω(state[app2.DesiredState(2).StoreKey()]).Should(EqualDesiredState(app2.DesiredState(2)))
				Ω(state[app3.DesiredState(1).StoreKey()]).Should(EqualDesiredState(app3.DesiredState(1)))
			})
		})

	})

	Describe("Actual State", func() {
		var heartbeat models.Heartbeat
		Describe("Writing and reading actual state", func() {
			var app3 appfixture.AppFixture
			BeforeEach(func() {
				app3 = dea.GetApp(2)

				heartbeat = dea.HeartbeatWith(app1.InstanceAtIndex(0).Heartbeat(), app2.InstanceAtIndex(1).Heartbeat(), app3.InstanceAtIndex(0).Heartbeat())
				err := store.SyncHeartbeats(heartbeat)
				Ω(err).ShouldNot(HaveOccured())
			})

			It("should return the stored actual state", func() {
				state, err := store.GetInstanceHeartbeats()
				Ω(err).ShouldNot(HaveOccured())
				Ω(state).Should(HaveLen(3))

				Ω(state).Should(ContainElement(app1.InstanceAtIndex(0).Heartbeat()))
				Ω(state).Should(ContainElement(app2.InstanceAtIndex(1).Heartbeat()))
				Ω(state).Should(ContainElement(app3.InstanceAtIndex(0).Heartbeat()))
			})

			It("should also return the state queried by individual app", func() {
				state, err := store.GetInstanceHeartbeatsForApp(app1.AppGuid, app1.AppVersion)
				Ω(err).ShouldNot(HaveOccured())
				Ω(state).Should(ContainElement(app1.InstanceAtIndex(0).Heartbeat()))
			})

			Context("when the app is not present", func() {
				It("should also return the state queried by individual app", func() {
					state, err := store.GetInstanceHeartbeatsForApp("abc", "def")
					Ω(err).ShouldNot(HaveOccured())
					Ω(state).Should(BeEmpty())
				})
			})

			Context("when the TTL expires", func() {
				BeforeEach(func() {
					timeProvider.IncrementBySeconds(conf.HeartbeatTTL())
				})

				It("should expire the nodes appropriately", func() {
					state, err := store.GetInstanceHeartbeats()
					Ω(err).ShouldNot(HaveOccured())
					Ω(state).Should(HaveLen(0))
				})
			})

			Describe("Updating Actual state", func() {
				BeforeEach(func() {
					timeProvider.IncrementBySeconds(conf.HeartbeatTTL() - 10)

					heartbeat = dea.HeartbeatWith(app1.InstanceAtIndex(0).Heartbeat(), app2.InstanceAtIndex(1).Heartbeat())
					heartbeat.InstanceHeartbeats[1].State = models.InstanceStateCrashed
					err := store.SyncHeartbeats(heartbeat)

					Ω(err).ShouldNot(HaveOccured())
				})

				It("should update the correct entry and delete any missing entries", func() {
					state, err := store.GetInstanceHeartbeats()
					Ω(err).ShouldNot(HaveOccured())
					Ω(state).Should(HaveLen(2))

					Ω(state).Should(ContainElement(app1.InstanceAtIndex(0).Heartbeat()))
					Ω(state).Should(ContainElement(heartbeat.InstanceHeartbeats[1]))
				})

				It("should bump the TTL", func() {
					timeProvider.IncrementBySeconds(10)
					state, err := store.GetInstanceHeartbeats()
					Ω(err).ShouldNot(HaveOccured())
					Ω(state).Should(HaveLen(2))
					Ω(state).Should(ContainElement(app1.InstanceAtIndex(0).Heartbeat()))
					Ω(state).Should(ContainElement(heartbeat.InstanceHeartbeats[1]))
				})
			})
		})
	})

	Describe("Crash State", func() {
		BeforeEach(func() {
			crashCount1 = models.CrashCount{
				AppGuid:       "foo",
				AppVersion:    "123",
				InstanceIndex: 0,
				CrashCount:    2,
				CreatedAt:     1,
			}
			crashCount2 = models.CrashCount{
				AppGuid:       "foo",
				AppVersion:    "123",
				InstanceIndex: 1,
				CrashCount:    1,
				CreatedAt:     3,
			}
		})

		Describe("Writing and reading crash counts", func() {
			BeforeEach(func() {
				err := store.SaveCrashCounts(crashCount1, crashCount2)
				Ω(err).ShouldNot(HaveOccured())
			})

			It("should return the stored counts", func() {
				state, err := store.GetCrashCounts()
				Ω(err).ShouldNot(HaveOccured())
				Ω(state).Should(HaveLen(2))

				Ω(state[crashCount1.StoreKey()]).Should(Equal(crashCount1))
				Ω(state[crashCount2.StoreKey()]).Should(Equal(crashCount2))
			})

			Context("when the TTL expires", func() {
				BeforeEach(func() {
					timeProvider.IncrementBySeconds(uint64(conf.MaximumBackoffDelay().Seconds()) * 2)
				})

				It("should expire the nodes appropriately", func() {
					state, err := store.GetCrashCounts()
					Ω(err).ShouldNot(HaveOccured())
					Ω(state).Should(HaveLen(0))
				})
			})

			Describe("Updating Crash state", func() {
				BeforeEach(func() {
					timeProvider.IncrementBySeconds(uint64(conf.MaximumBackoffDelay().Seconds())*2 - 10)
					crashCount2.CrashCount += 1
					err := store.SaveCrashCounts(crashCount2)
					Ω(err).ShouldNot(HaveOccured())
				})

				It("should update the correct entry", func() {
					state, err := store.GetCrashCounts()
					Ω(err).ShouldNot(HaveOccured())
					Ω(state).Should(HaveLen(2))

					Ω(state[crashCount1.StoreKey()]).Should(Equal(crashCount1))
					Ω(state[crashCount2.StoreKey()]).Should(Equal(crashCount2))
				})

				It("should bump the TTL", func() {
					timeProvider.IncrementBySeconds(10)
					state, err := store.GetCrashCounts()
					Ω(err).ShouldNot(HaveOccured())
					Ω(state).Should(HaveLen(1))
					Ω(state[crashCount2.StoreKey()]).Should(Equal(crashCount2))
				})
			})
		})
	})

	Describe("Pending Start Messages", func() {
		BeforeEach(func() {
			startMessage1 = models.NewPendingStartMessage(timeProvider.Time(), 10, 4, "ABC", "123", 1, 1.0, models.PendingStartMessageReasonMissing)
			startMessage2 = models.NewPendingStartMessage(timeProvider.Time(), 10, 4, "DEF", "456", 1, 1.0, models.PendingStartMessageReasonMissing)
		})

		Describe("Writing and reading pending start messages", func() {
			BeforeEach(func() {
				err := store.SavePendingStartMessages(startMessage1, startMessage2)
				Ω(err).ShouldNot(HaveOccured())
			})

			It("should return the pending start messages", func() {
				messages, err := store.GetPendingStartMessages()
				Ω(err).ShouldNot(HaveOccured())
				Ω(messages).Should(HaveLen(2))
				Ω(messages[startMessage1.StoreKey()]).Should(Equal(startMessage1))
				Ω(messages[startMessage2.StoreKey()]).Should(Equal(startMessage2))
			})

			Describe("Updating pending start messages", func() {
				It("should update the correct message", func() {
					startMessage2.Priority = 0.7
					err := store.SavePendingStartMessages(startMessage2)
					Ω(err).ShouldNot(HaveOccured())

					messages, err := store.GetPendingStartMessages()
					Ω(err).ShouldNot(HaveOccured())
					Ω(messages).Should(HaveLen(2))
					Ω(messages[startMessage1.StoreKey()]).Should(Equal(startMessage1))
					Ω(messages[startMessage2.StoreKey()]).Should(Equal(startMessage2))

				})
			})

			Describe("Deleting pending start messages", func() {
				It("should delete the specified message but not the others", func() {
					err := store.DeletePendingStartMessages(startMessage1)
					Ω(err).ShouldNot(HaveOccured())

					messages, err := store.GetPendingStartMessages()
					Ω(err).ShouldNot(HaveOccured())
					Ω(messages).Should(HaveLen(1))
					Ω(messages[startMessage2.StoreKey()]).Should(Equal(startMessage2))
				})
			})
		})
	})

	Describe("Pending Stop Messages", func() {
		BeforeEach(func() {
			stopMessage1 = models.NewPendingStopMessage(timeProvider.Time(), 10, 4, "ABC", "123", "XYZ", models.PendingStopMessageReasonExtra)
			stopMessage2 = models.NewPendingStopMessage(timeProvider.Time(), 10, 4, "DEF", "456", "ALPHA", models.PendingStopMessageReasonExtra)
		})

		Describe("Writing and reading pending stop messages", func() {
			BeforeEach(func() {
				err := store.SavePendingStopMessages(stopMessage1, stopMessage2)
				Ω(err).ShouldNot(HaveOccured())
			})

			It("should return the pending stop messages", func() {
				messages, err := store.GetPendingStopMessages()
				Ω(err).ShouldNot(HaveOccured())
				Ω(messages).Should(HaveLen(2))
				Ω(messages[stopMessage1.StoreKey()]).Should(Equal(stopMessage1))
				Ω(messages[stopMessage2.StoreKey()]).Should(Equal(stopMessage2))
			})

			Describe("Updating pending stop messages", func() {
				It("should update the correct message", func() {
					stopMessage2.SendOn += 10
					err := store.SavePendingStopMessages(stopMessage2)
					Ω(err).ShouldNot(HaveOccured())

					messages, err := store.GetPendingStopMessages()
					Ω(err).ShouldNot(HaveOccured())
					Ω(messages).Should(HaveLen(2))
					Ω(messages[stopMessage1.StoreKey()]).Should(Equal(stopMessage1))
					Ω(messages[stopMessage2.StoreKey()]).Should(Equal(stopMessage2))

				})
			})

			Describe("Deleting pending stop messages", func() {
				It("should delete the specified message but not the others", func() {
					err := store.DeletePendingStopMessages(stopMessage1)
					Ω(err).ShouldNot(HaveOccured())

					messages, err := store.GetPendingStopMessages()
					Ω(err).ShouldNot(HaveOccured())
					Ω(messages).Should(HaveLen(1))
					Ω(messages[stopMessage2.StoreKey()]).Should(Equal(stopMessage2))
				})
			})
		})
	})

	Describe("Freshness", func() {
		Describe("Desired freshness", func() {
			Context("when the desired freshness is missing", func() {
				Context("and we bump the freshnesss", func() {
					BeforeEach(func() {
						err := store.BumpDesiredFreshness(timeProvider.Time())
						Ω(err).ShouldNot(HaveOccured())
					})

					It("should mark the desired state as fresh", func() {
						isFresh, err := store.IsDesiredStateFresh()
						Ω(err).ShouldNot(HaveOccured())
						Ω(isFresh).Should(BeTrue())
					})

					Context("when the desired state TTL expires", func() {
						BeforeEach(func() {
							timeProvider.IncrementBySeconds(conf.DesiredFreshnessTTL())
						})

						It("should no longer be fresh", func() {
							isFresh, err := store.IsDesiredStateFresh()
							Ω(err).ShouldNot(HaveOccured())
							Ω(isFresh).Should(BeFalse())
						})
					})
				})

				It("should not be fresh", func() {
					isFresh, err := store.IsDesiredStateFresh()
					Ω(err).ShouldNot(HaveOccured())
					Ω(isFresh).Should(BeFalse())
				})
			})

			Context("when the desired freshness is present", func() {
				BeforeEach(func() {
					timeProvider.IncrementBySeconds(10)
					err := store.BumpDesiredFreshness(timeProvider.Time())
					Ω(err).ShouldNot(HaveOccured())

				})

				It("should bump the ttl", func() {
					timeProvider.IncrementBySeconds(conf.DesiredFreshnessTTL() - 10)

					isFresh, err := store.IsDesiredStateFresh()
					Ω(err).ShouldNot(HaveOccured())
					Ω(isFresh).Should(BeTrue())
				})

				It("should expire after the new ttl expires", func() {
					timeProvider.IncrementBySeconds(conf.DesiredFreshnessTTL())

					isFresh, err := store.IsDesiredStateFresh()
					Ω(err).ShouldNot(HaveOccured())
					Ω(isFresh).Should(BeFalse())
				})
			})
		})

		Describe("Bumping actual freshness", func() {
			Context("when the actual freshness is missing", func() {
				Context("and we bump the freshnesss", func() {
					BeforeEach(func() {
						err := store.BumpActualFreshness(timeProvider.Time())
						Ω(err).ShouldNot(HaveOccured())
					})

					It("should not report the actual state as fresh", func() {
						isFresh, err := store.IsActualStateFresh(timeProvider.Time())
						Ω(err).ShouldNot(HaveOccured())
						Ω(isFresh).Should(BeFalse())
					})

					Context("when we bump the freshness again before expiry", func() {
						BeforeEach(func() {
							timeProvider.IncrementBySeconds(10)
							err := store.BumpActualFreshness(timeProvider.Time())
							Ω(err).ShouldNot(HaveOccured())
							timeProvider.IncrementBySeconds(conf.ActualFreshnessTTL() - 10)
						})

						It("should be fresh", func() {
							isFresh, err := store.IsActualStateFresh(timeProvider.Time())
							Ω(err).ShouldNot(HaveOccured())
							Ω(isFresh).Should(BeTrue())
						})

						Context("when we run past expiration time", func() {
							BeforeEach(func() {
								timeProvider.IncrementBySeconds(10)
							})

							It("should no longer be fresh", func() {
								isFresh, err := store.IsActualStateFresh(timeProvider.Time())
								Ω(err).ShouldNot(HaveOccured())
								Ω(isFresh).Should(BeFalse())
							})

							Context("When we start bumping freshness again", func() {
								BeforeEach(func() {
									err := store.BumpActualFreshness(timeProvider.Time())
									Ω(err).ShouldNot(HaveOccured())
									timeProvider.IncrementBySeconds(10)
									err = store.BumpActualFreshness(timeProvider.Time())
									Ω(err).ShouldNot(HaveOccured())
									timeProvider.IncrementBySeconds(conf.ActualFreshnessTTL() - 10)
								})

								It("should become fresh again", func() {
									isFresh, err := store.IsActualStateFresh(timeProvider.Time())
									Ω(err).ShouldNot(HaveOccured())
									Ω(isFresh).Should(BeTrue())
								})
							})
						})
					})
				})

				It("should not be fresh", func() {
					isFresh, err := store.IsActualStateFresh(timeProvider.Time())
					Ω(err).ShouldNot(HaveOccured())
					Ω(isFresh).Should(BeFalse())
				})
			})
		})

		Describe("VerifyFreshness", func() {
			Context("when both desired and actual are not fresh", func() {
				It("should return the correct error", func() {
					Ω(store.VerifyFreshness(timeProvider.Time())).Should(Equal(storepackage.ActualAndDesiredAreNotFreshError))
				})
			})

			Context("when only desired is fresh", func() {
				BeforeEach(func() {
					store.BumpDesiredFreshness(timeProvider.Time())
				})

				It("should return the correct error", func() {
					Ω(store.VerifyFreshness(timeProvider.Time())).Should(Equal(storepackage.ActualIsNotFreshError))
				})
			})

			Context("when only actual is fresh", func() {
				BeforeEach(func() {
					store.BumpActualFreshness(timeProvider.Time())
					timeProvider.IncrementBySeconds(10)
					store.BumpActualFreshness(timeProvider.Time())
					timeProvider.IncrementBySeconds(conf.ActualFreshnessTTL() - 10)
				})

				It("should return the correct error", func() {
					Ω(store.VerifyFreshness(timeProvider.Time())).Should(Equal(storepackage.DesiredIsNotFreshError))
				})
			})

			Context("when both desired and actual are fresh", func() {
				BeforeEach(func() {
					store.BumpDesiredFreshness(timeProvider.Time())
					store.BumpActualFreshness(timeProvider.Time())
					timeProvider.IncrementBySeconds(10)
					store.BumpActualFreshness(timeProvider.Time())
					timeProvider.IncrementBySeconds(conf.ActualFreshnessTTL() - 10)
				})

				It("should not error", func() {
					Ω(store.VerifyFreshness(timeProvider.Time())).Should(BeNil())
				})
			})
		})
	})

	Describe("Getting Apps", func() {
		var app3, app4 appfixture.AppFixture
		BeforeEach(func() {

			//4 apps: app1 has desired, actual & crashes, app2 has desired only, app3 has actual only, app4 has crashes only
			app3 = dea.GetApp(2)
			app4 = dea.GetApp(3)

			crashCount1 = models.CrashCount{
				AppGuid:       app1.AppGuid,
				AppVersion:    app1.AppVersion,
				InstanceIndex: 0,
				CrashCount:    2,
				CreatedAt:     1,
			}
			crashCount2 = models.CrashCount{
				AppGuid:       app4.AppGuid,
				AppVersion:    app4.AppVersion,
				InstanceIndex: 1,
				CrashCount:    1,
				CreatedAt:     4,
			}

			store.SyncDesiredState(app1.DesiredState(1), app2.DesiredState(3))
			store.SyncHeartbeats(dea.HeartbeatWith(app1.InstanceAtIndex(0).Heartbeat(), app3.InstanceAtIndex(2).Heartbeat()))
			store.SaveCrashCounts(crashCount1, crashCount2)

		})

		Describe("GetApp()", func() {
			Context("when the app has actual & desired state", func() {
				It("should return the app", func() {
					app, err := store.GetApp(app1.AppGuid, app1.AppVersion)
					Ω(err).ShouldNot(HaveOccured())

					Ω(app.AppGuid).Should(Equal(app1.AppGuid))
					Ω(app.AppVersion).Should(Equal(app1.AppVersion))
					Ω(app.Desired).Should(EqualDesiredState(app1.DesiredState(1)))
					Ω(app.InstanceHeartbeats).Should(ContainElement(app1.InstanceAtIndex(0).Heartbeat()))
					Ω(app.CrashCounts[0]).Should(Equal(crashCount1))
				})
			})

			Context("when the app has desired state only", func() {
				It("should return the app", func() {
					app, err := store.GetApp(app2.AppGuid, app2.AppVersion)
					Ω(err).ShouldNot(HaveOccured())

					Ω(app.AppGuid).Should(Equal(app2.AppGuid))
					Ω(app.AppVersion).Should(Equal(app2.AppVersion))
					Ω(app.Desired).Should(EqualDesiredState(app2.DesiredState(3)))
					Ω(app.InstanceHeartbeats).Should(BeEmpty())
					Ω(app.CrashCounts).Should(BeEmpty())
				})
			})

			Context("when the app has actual state only", func() {
				It("should return the app", func() {
					app, err := store.GetApp(app3.AppGuid, app3.AppVersion)
					Ω(err).ShouldNot(HaveOccured())

					Ω(app.AppGuid).Should(Equal(app3.AppGuid))
					Ω(app.AppVersion).Should(Equal(app3.AppVersion))
					Ω(app.Desired).Should(BeZero())
					Ω(app.InstanceHeartbeats).Should(ContainElement(app3.InstanceAtIndex(2).Heartbeat()))
					Ω(app.CrashCounts).Should(BeEmpty())
				})
			})

			Context("when the app has crash counts only", func() {
				It("should return nil and report that the app was not found", func() {
					app, err := store.GetApp(app4.AppGuid, app4.AppVersion)
					Ω(app).Should(BeNil())
					Ω(err).Should(Equal(storepackage.AppNotFoundError))
				})
			})

			Context("when the app is not present", func() {
				It("should return nil and report that the app was not found", func() {
					app, err := store.GetApp("no guid!", "0.0.0")
					Ω(app).Should(BeNil())
					Ω(err).Should(Equal(storepackage.AppNotFoundError))
				})
			})
		})

		Describe("GetApps()", func() {
			It("should return a hash for any apps that have actual and/or desired state", func() {
				apps, err := store.GetApps()
				Ω(err).ShouldNot(HaveOccured())
				Ω(apps).Should(HaveLen(3))

				for _, appFixture := range []appfixture.AppFixture{app1, app2, app3} {
					app, err := store.GetApp(appFixture.AppGuid, appFixture.AppVersion)
					Ω(err).ShouldNot(HaveOccured())
					key := store.AppKey(app.AppGuid, app.AppVersion)
					Ω(apps[key].AppGuid).Should(Equal(app.AppGuid))
					Ω(apps[key].AppVersion).Should(Equal(app.AppVersion))
					Ω(apps[key].Desired).Should(EqualDesiredState(app.Desired))
					Ω(apps[key].InstanceHeartbeats).Should(Equal(app.InstanceHeartbeats))
					Ω(apps[key].CrashCounts).Should(Equal(app.CrashCounts))
				}
			})
		})
	})

	Describe("Metrics", func() {
		Describe("Getting and setting a metric", func() {
			BeforeEach(func() {
				err := store.SaveMetric("sprockets", 17)
				Ω(err).ShouldNot(HaveOccured())
			})

			Context("when the metric is present", func() {
				It("should return the stored value and no error", func() {
					value, err := store.GetMetric("sprockets")
					Ω(err).ShouldNot(HaveOccured())
					Ω(value).Should(BeNumerically("==", 17))
				})

				Context("and it is overwritten", func() {
					BeforeEach(func() {
						err := store.SaveMetric("sprockets", 23.5)
						Ω(err).ShouldNot(HaveOccured())
					})

					It("should return the new value", func() {
						value, err := store.GetMetric("sprockets")
						Ω(err).ShouldNot(HaveOccured())
						Ω(value).Should(BeNumerically("==", 23.5))
					})
				})
			})

			Context("when the metric is not present", func() {
				It("should return -1 and an error", func() {
					value, err := store.GetMetric("nonexistent")
					Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
					Ω(value).Should(BeNumerically("==", -1))
				})
			})
		})
	})

})
