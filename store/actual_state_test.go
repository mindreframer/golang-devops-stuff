package store_test

import (
	"github.com/cloudfoundry/hm9000/helpers/workerpool"
	. "github.com/cloudfoundry/hm9000/store"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
)

var _ = Describe("Actual State", func() {
	var (
		store        Store
		storeAdapter storeadapter.StoreAdapter
		conf         *config.Config
		dea          appfixture.DeaFixture
		otherDea     appfixture.DeaFixture
	)

	BeforeEach(func() {
		var err error
		conf, err = config.DefaultConfig()
		Ω(err).ShouldNot(HaveOccurred())
		storeAdapter = storeadapter.NewETCDStoreAdapter(etcdRunner.NodeURLS(), workerpool.NewWorkerPool(conf.StoreMaxConcurrentRequests))
		err = storeAdapter.Connect()
		Ω(err).ShouldNot(HaveOccurred())
		conf.StoreHeartbeatCacheRefreshIntervalInMilliseconds = 100
		store = NewStore(conf, storeAdapter, fakelogger.NewFakeLogger())

		dea = appfixture.NewDeaFixture()
		otherDea = appfixture.NewDeaFixture()
	})

	AfterEach(func() {
		storeAdapter.Disconnect()
	})

	Describe("Saving actual state", func() {
		BeforeEach(func() {
			store.SyncHeartbeats(dea.HeartbeatWith(
				dea.GetApp(0).InstanceAtIndex(1).Heartbeat(),
				dea.GetApp(1).InstanceAtIndex(3).Heartbeat(),
			))
		})

		It("should save the instance heartbeats for the passed-in heartbeat", func() {
			results, err := store.GetInstanceHeartbeats()
			Ω(err).ShouldNot(HaveOccurred())
			Ω(results).Should(HaveLen(2))
			Ω(results).Should(ContainElement(dea.GetApp(0).InstanceAtIndex(1).Heartbeat()))
			Ω(results).Should(ContainElement(dea.GetApp(1).InstanceAtIndex(3).Heartbeat()))
		})

		Context("when there are already instance heartbeats stored for the DEA in question", func() {
			var modifiedHeartbeat models.InstanceHeartbeat
			BeforeEach(func() {
				modifiedHeartbeat = dea.GetApp(1).InstanceAtIndex(3).Heartbeat()
				modifiedHeartbeat.State = models.InstanceStateEvacuating
				store.SyncHeartbeats(dea.HeartbeatWith(
					modifiedHeartbeat,
					dea.GetApp(2).InstanceAtIndex(2).Heartbeat(),
				))
			})

			It("should sync the heartbeats (add new ones, adjust ones that have changed state, and delete old ones)", func() {
				results, err := store.GetInstanceHeartbeats()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(results).Should(HaveLen(2))
				Ω(results).Should(ContainElement(modifiedHeartbeat))
				Ω(results).Should(ContainElement(dea.GetApp(2).InstanceAtIndex(2).Heartbeat()))
			})
		})

		Context("when saving multiple heartbeats at once", func() {
			var modifiedHeartbeat models.InstanceHeartbeat
			var yetAnotherDea appfixture.DeaFixture

			BeforeEach(func() {
				yetAnotherDea = appfixture.NewDeaFixture()

				store.SyncHeartbeats(dea.HeartbeatWith(
					dea.GetApp(0).InstanceAtIndex(1).Heartbeat(),
					dea.GetApp(1).InstanceAtIndex(3).Heartbeat(),
				), otherDea.HeartbeatWith(
					otherDea.GetApp(3).InstanceAtIndex(0).Heartbeat(),
					otherDea.GetApp(2).InstanceAtIndex(1).Heartbeat(),
				), yetAnotherDea.HeartbeatWith(
					yetAnotherDea.GetApp(0).InstanceAtIndex(0).Heartbeat(),
				))

				modifiedHeartbeat = dea.GetApp(1).InstanceAtIndex(3).Heartbeat()
				modifiedHeartbeat.State = models.InstanceStateEvacuating
				store.SyncHeartbeats(dea.HeartbeatWith(
					modifiedHeartbeat,
					dea.GetApp(2).InstanceAtIndex(2).Heartbeat(),
				), otherDea.HeartbeatWith(
					otherDea.GetApp(2).InstanceAtIndex(1).Heartbeat(),
					otherDea.GetApp(3).InstanceAtIndex(2).Heartbeat(),
				))
			})

			It("should work", func() {
				results, err := store.GetInstanceHeartbeats()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(results).Should(HaveLen(5))
				Ω(results).Should(ContainElement(modifiedHeartbeat))
				Ω(results).Should(ContainElement(dea.GetApp(2).InstanceAtIndex(2).Heartbeat()))
				Ω(results).Should(ContainElement(otherDea.GetApp(2).InstanceAtIndex(1).Heartbeat()))
				Ω(results).Should(ContainElement(otherDea.GetApp(3).InstanceAtIndex(2).Heartbeat()))
				Ω(results).Should(ContainElement(yetAnotherDea.GetApp(0).InstanceAtIndex(0).Heartbeat()))
			})
		})

		Context("when one of the keys fails to delete", func() {
			It("should soldier on", func() {
				store.SyncHeartbeats(dea.HeartbeatWith(
					dea.GetApp(0).InstanceAtIndex(1).Heartbeat(),
					dea.GetApp(1).InstanceAtIndex(3).Heartbeat(),
				))

				done := make(chan error, 2)

				go func() {
					done <- store.SyncHeartbeats(dea.HeartbeatWith(
						dea.GetApp(0).InstanceAtIndex(1).Heartbeat(),
					))
				}()

				go func() {
					done <- store.SyncHeartbeats(dea.HeartbeatWith(
						dea.GetApp(0).InstanceAtIndex(1).Heartbeat(),
					))
				}()

				err1 := <-done
				err2 := <-done
				Ω(err1).ShouldNot(HaveOccurred())
				Ω(err2).ShouldNot(HaveOccurred())
			})
		})

		Context("when something goes wrong and the in-memory cache no longer matches the store", func() {
			It("should eventually recover", func() {
				//Delete one of the heartbeats
				corruptedHeartbeat := dea.GetApp(0).InstanceAtIndex(1).Heartbeat()
				storeAdapter.Delete("/hm/v1/apps/actual/" + store.AppKey(corruptedHeartbeat.AppGuid, corruptedHeartbeat.AppVersion) + "/" + corruptedHeartbeat.InstanceGuid)

				//See that it's gone
				results, err := store.GetInstanceHeartbeats()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(results).Should(HaveLen(1))

				//Try to put it back
				store.SyncHeartbeats(dea.HeartbeatWith(
					dea.GetApp(0).InstanceAtIndex(1).Heartbeat(),
					dea.GetApp(1).InstanceAtIndex(3).Heartbeat(),
				))

				//See that we didn't... because it's still in the cache...
				results, err = store.GetInstanceHeartbeats()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(results).Should(HaveLen(1))

				//Eventually the cache should be reloaded...
				Eventually(func() []models.InstanceHeartbeat {
					store.SyncHeartbeats(dea.HeartbeatWith(
						dea.GetApp(0).InstanceAtIndex(1).Heartbeat(),
						dea.GetApp(1).InstanceAtIndex(3).Heartbeat(),
					))

					results, err = store.GetInstanceHeartbeats()
					Ω(err).ShouldNot(HaveOccurred())
					return results
				}, 1.0, 0.05).Should(HaveLen(2)) //...and the heartbeat should return
			})
		})
	})

	Describe("Fetching all actual state", func() {
		Context("when there is none saved", func() {
			It("should come back empty", func() {
				results, err := store.GetInstanceHeartbeats()
				Ω(err).ShouldNot(HaveOccurred())
				Ω(results).Should(BeEmpty())
			})
		})

		Context("when there is actual state saved", func() {
			var heartbeatOnDea, heartbeatOnOtherDea models.InstanceHeartbeat

			BeforeEach(func() {
				appOnBothDeas := appfixture.NewAppFixture()

				heartbeatOnDea = appOnBothDeas.InstanceAtIndex(0).Heartbeat()
				heartbeatOnDea.DeaGuid = dea.DeaGuid

				heartbeatOnOtherDea = appOnBothDeas.InstanceAtIndex(1).Heartbeat()
				heartbeatOnOtherDea.DeaGuid = otherDea.DeaGuid

				store.SyncHeartbeats(dea.HeartbeatWith(
					dea.GetApp(0).InstanceAtIndex(1).Heartbeat(),
					dea.GetApp(1).InstanceAtIndex(3).Heartbeat(),
					heartbeatOnDea,
				))

				store.SyncHeartbeats(otherDea.HeartbeatWith(
					otherDea.GetApp(0).InstanceAtIndex(1).Heartbeat(),
					otherDea.GetApp(1).InstanceAtIndex(0).Heartbeat(),
					heartbeatOnOtherDea,
				))
			})

			Context("when the DEA heartbeats have not expired", func() {
				It("should return the instance heartbeats", func() {
					results, err := store.GetInstanceHeartbeats()
					Ω(err).ShouldNot(HaveOccurred())
					Ω(results).Should(HaveLen(6))
					Ω(results).Should(ContainElement(dea.GetApp(0).InstanceAtIndex(1).Heartbeat()))
					Ω(results).Should(ContainElement(dea.GetApp(1).InstanceAtIndex(3).Heartbeat()))
					Ω(results).Should(ContainElement(heartbeatOnDea))
					Ω(results).Should(ContainElement(otherDea.GetApp(0).InstanceAtIndex(1).Heartbeat()))
					Ω(results).Should(ContainElement(otherDea.GetApp(1).InstanceAtIndex(0).Heartbeat()))
					Ω(results).Should(ContainElement(heartbeatOnOtherDea))
				})
			})

			Context("when a DEA heartbeat has expired", func() {
				BeforeEach(func() {
					storeAdapter.Delete("/hm/v1/dea-presence/" + dea.DeaGuid)
				})

				It("should not return any expired instance heartbeats", func() {
					results, err := store.GetInstanceHeartbeats()
					Ω(err).ShouldNot(HaveOccurred())
					Ω(results).Should(HaveLen(3))
					Ω(results).Should(ContainElement(otherDea.GetApp(0).InstanceAtIndex(1).Heartbeat()))
					Ω(results).Should(ContainElement(otherDea.GetApp(1).InstanceAtIndex(0).Heartbeat()))
					Ω(results).Should(ContainElement(heartbeatOnOtherDea))

					//we fetch twice to ensure that nothing is incorrectly deleted
					results, err = store.GetInstanceHeartbeats()
					Ω(err).ShouldNot(HaveOccurred())
					Ω(results).Should(HaveLen(3))
					Ω(results).Should(ContainElement(otherDea.GetApp(0).InstanceAtIndex(1).Heartbeat()))
					Ω(results).Should(ContainElement(otherDea.GetApp(1).InstanceAtIndex(0).Heartbeat()))
					Ω(results).Should(ContainElement(heartbeatOnOtherDea))
				})

				It("should remove expired instance heartbeats from the store", func() {
					_, err := storeAdapter.Get("/hm/v1/apps/actual/" + store.AppKey(dea.GetApp(0).AppGuid, dea.GetApp(0).AppVersion) + "/" + dea.GetApp(0).InstanceAtIndex(1).Heartbeat().StoreKey())
					Ω(err).ShouldNot(HaveOccurred())
					_, err = storeAdapter.Get("/hm/v1/apps/actual/" + store.AppKey(dea.GetApp(1).AppGuid, dea.GetApp(1).AppVersion) + "/" + dea.GetApp(1).InstanceAtIndex(3).Heartbeat().StoreKey())
					Ω(err).ShouldNot(HaveOccurred())

					_, err = store.GetInstanceHeartbeats()
					Ω(err).ShouldNot(HaveOccurred())

					_, err = storeAdapter.Get("/hm/v1/apps/actual/" + store.AppKey(dea.GetApp(0).AppGuid, dea.GetApp(0).AppVersion) + "/" + dea.GetApp(0).InstanceAtIndex(1).Heartbeat().StoreKey())
					Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
					_, err = storeAdapter.Get("/hm/v1/apps/actual/" + store.AppKey(dea.GetApp(1).AppGuid, dea.GetApp(1).AppVersion) + "/" + dea.GetApp(1).InstanceAtIndex(3).Heartbeat().StoreKey())
					Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
				})

				Context("if it fails to remove them", func() {
					It("should soldier on", func() {
						resultChan := make(chan []models.InstanceHeartbeat, 2)
						errChan := make(chan error, 2)
						go func() {
							results, err := store.GetInstanceHeartbeats()
							resultChan <- results
							errChan <- err
						}()

						go func() {
							results, err := store.GetInstanceHeartbeats()
							resultChan <- results
							errChan <- err
						}()

						Ω(<-resultChan).Should(HaveLen(3))
						Ω(<-resultChan).Should(HaveLen(3))
						Ω(<-errChan).ShouldNot(HaveOccurred())
						Ω(<-errChan).ShouldNot(HaveOccurred())
					})
				})
			})
		})
	})

	Describe("Fetching actual state for a specific app guid & version", func() {
		var app appfixture.AppFixture
		BeforeEach(func() {
			app = appfixture.NewAppFixture()
		})

		Context("when there is none saved", func() {
			It("should come back empty", func() {
				results, err := store.GetInstanceHeartbeatsForApp(app.AppGuid, app.AppVersion)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(results).Should(BeEmpty())
			})
		})

		Context("when there is actual state saved", func() {
			var heartbeatA, heartbeatB models.InstanceHeartbeat

			BeforeEach(func() {
				heartbeatA = app.InstanceAtIndex(0).Heartbeat()
				heartbeatA.DeaGuid = "A"

				store.SyncHeartbeats(models.Heartbeat{
					DeaGuid: "A",
					InstanceHeartbeats: []models.InstanceHeartbeat{
						heartbeatA,
					},
				})

				heartbeatB = app.InstanceAtIndex(1).Heartbeat()
				heartbeatB.DeaGuid = "B"

				store.SyncHeartbeats(models.Heartbeat{
					DeaGuid: "B",
					InstanceHeartbeats: []models.InstanceHeartbeat{
						heartbeatB,
					},
				})
			})

			Context("when the corresponding DEA heartbeat has not expired", func() {
				It("should return the instance heartbeats", func() {
					results, err := store.GetInstanceHeartbeatsForApp(app.AppGuid, app.AppVersion)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(results).Should(HaveLen(2))
					Ω(results).Should(ContainElement(heartbeatA))
					Ω(results).Should(ContainElement(heartbeatB))
				})
			})

			Context("when an associated DEA heartbeat has expired", func() {
				BeforeEach(func() {
					storeAdapter.Delete("/hm/v1/dea-presence/A")
				})

				It("should not return any expired instance heartbeats", func() {
					results, err := store.GetInstanceHeartbeatsForApp(app.AppGuid, app.AppVersion)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(results).Should(HaveLen(1))
					Ω(results).Should(ContainElement(heartbeatB))
				})

				It("should remove expired instance heartbeats from the store", func() {
					_, err := storeAdapter.Get("/hm/v1/apps/actual/" + store.AppKey(app.AppGuid, app.AppVersion) + "/" + heartbeatA.StoreKey())
					Ω(err).ShouldNot(HaveOccurred())

					_, err = store.GetInstanceHeartbeatsForApp(app.AppGuid, app.AppVersion)
					Ω(err).ShouldNot(HaveOccurred())

					_, err = storeAdapter.Get("/hm/v1/apps/actual/" + store.AppKey(app.AppGuid, app.AppVersion) + "/" + heartbeatA.StoreKey())
					Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))
				})

				Context("if it fails to remove them", func() {
					It("should soldier on", func() {
						resultChan := make(chan []models.InstanceHeartbeat, 2)
						errChan := make(chan error, 2)
						go func() {
							results, err := store.GetInstanceHeartbeatsForApp(app.AppGuid, app.AppVersion)
							resultChan <- results
							errChan <- err
						}()

						go func() {
							results, err := store.GetInstanceHeartbeatsForApp(app.AppGuid, app.AppVersion)
							resultChan <- results
							errChan <- err
						}()

						Ω(<-resultChan).Should(HaveLen(1))
						Ω(<-resultChan).Should(HaveLen(1))
						Ω(<-errChan).ShouldNot(HaveOccurred())
						Ω(<-errChan).ShouldNot(HaveOccurred())
					})
				})
			})

			Context("when all the DEA heartbeats have expired", func() {
				BeforeEach(func() {
					storeAdapter.Delete("/hm/v1/dea-presence/A", "/hm/v1/dea-presence/B")
				})

				It("should not return any instance heartbeats", func() {
					results, err := store.GetInstanceHeartbeatsForApp(app.AppGuid, app.AppVersion)
					Ω(err).ShouldNot(HaveOccurred())
					Ω(results).ShouldNot(BeNil())
					Ω(results).Should(HaveLen(0))
				})
			})
		})
	})
})
