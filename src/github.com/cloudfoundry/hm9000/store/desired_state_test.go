package store_test

import (
	. "github.com/cloudfoundry/hm9000/store"
	. "github.com/cloudfoundry/hm9000/testhelpers/custommatchers"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
)

var _ = Describe("Desired State", func() {
	var (
		store       Store
		etcdAdapter storeadapter.StoreAdapter
		conf        config.Config
		app1        appfixture.AppFixture
		app2        appfixture.AppFixture
		app3        appfixture.AppFixture
	)

	BeforeEach(func() {
		var err error
		conf, err = config.DefaultConfig()
		Ω(err).ShouldNot(HaveOccured())
		etcdAdapter = storeadapter.NewETCDStoreAdapter(etcdRunner.NodeURLS(), conf.StoreMaxConcurrentRequests)
		err = etcdAdapter.Connect()
		Ω(err).ShouldNot(HaveOccured())

		app1 = appfixture.NewAppFixture()
		app2 = appfixture.NewAppFixture()
		app3 = appfixture.NewAppFixture()

		store = NewStore(conf, etcdAdapter, fakelogger.NewFakeLogger())
	})

	AfterEach(func() {
		etcdAdapter.Disconnect()
	})

	Describe("Saving desired state", func() {
		BeforeEach(func() {
			err := store.SaveDesiredState(
				app1.DesiredState(1),
				app2.DesiredState(1),
			)
			Ω(err).ShouldNot(HaveOccured())
		})

		It("stores the passed in desired state", func() {
			node, err := etcdAdapter.Get("/apps/" + app1.AppGuid + "-" + app1.AppVersion + "/desired")
			Ω(err).ShouldNot(HaveOccured())
			Ω(node).Should(Equal(storeadapter.StoreNode{
				Key:   "/apps/" + app1.AppGuid + "-" + app1.AppVersion + "/desired",
				Value: app1.DesiredState(1).ToJSON(),
				TTL:   conf.DesiredStateTTL() - 1,
			}))
			node, err = etcdAdapter.Get("/apps/" + app2.AppGuid + "-" + app2.AppVersion + "/desired")
			Ω(err).ShouldNot(HaveOccured())
			Ω(node).Should(Equal(storeadapter.StoreNode{
				Key:   "/apps/" + app2.AppGuid + "-" + app2.AppVersion + "/desired",
				Value: app2.DesiredState(1).ToJSON(),
				TTL:   conf.DesiredStateTTL() - 1,
			}))
		})
	})

	Describe("Fetching desired state", func() {
		Context("When the desired state is present", func() {
			BeforeEach(func() {
				err := store.SaveDesiredState(
					app1.DesiredState(1),
					app2.DesiredState(1),
				)
				Ω(err).ShouldNot(HaveOccured())
			})

			It("can fetch the desired state", func() {
				desired, err := store.GetDesiredState()
				Ω(err).ShouldNot(HaveOccured())
				Ω(desired[app1.DesiredState(1).StoreKey()]).Should(EqualDesiredState(app1.DesiredState(1)))
				Ω(desired[app2.DesiredState(1).StoreKey()]).Should(EqualDesiredState(app2.DesiredState(1)))
			})
		})

		Context("when the desired state is empty", func() {
			BeforeEach(func() {
				err := store.SaveDesiredState(
					app1.DesiredState(1),
				)
				Ω(err).ShouldNot(HaveOccured())
				err = store.DeleteDesiredState(app1.DesiredState(1))
				Ω(err).ShouldNot(HaveOccured())
			})

			It("returns an empty array", func() {
				desired, err := store.GetDesiredState()
				Ω(err).ShouldNot(HaveOccured())
				Ω(desired).Should(BeEmpty())
			})
		})
	})

	Describe("Deleting desired state", func() {
		BeforeEach(func() {
			err := store.SaveDesiredState(
				app1.DesiredState(1),
				app2.DesiredState(1),
				app3.DesiredState(1),
			)
			Ω(err).ShouldNot(HaveOccured())
		})

		Context("When the desired state is present", func() {
			It("can delete the desired state (and only cares about the relevant fields)", func() {
				toDelete := []models.DesiredAppState{
					models.DesiredAppState{AppGuid: app1.AppGuid, AppVersion: app1.AppVersion},
					models.DesiredAppState{AppGuid: app3.AppGuid, AppVersion: app3.AppVersion},
				}
				err := store.DeleteDesiredState(toDelete...)
				Ω(err).ShouldNot(HaveOccured())

				desired, err := store.GetDesiredState()
				Ω(err).ShouldNot(HaveOccured())
				Ω(desired).Should(HaveLen(1))
				Ω(desired).Should(ContainElement(EqualDesiredState(app2.DesiredState(1))))
			})
		})

		Context("When the desired state key is not present", func() {
			It("returns an error, but does leave things in a broken state... for now...", func() {
				toDelete := []models.DesiredAppState{
					models.DesiredAppState{AppGuid: app1.AppGuid, AppVersion: app1.AppVersion},
					models.DesiredAppState{AppGuid: app3.AppGuid, AppVersion: app2.AppVersion}, //oops! this key is not present but we're trying to delete it
					models.DesiredAppState{AppGuid: app2.AppGuid, AppVersion: app2.AppVersion},
				}
				err := store.DeleteDesiredState(toDelete...)
				Ω(err).Should(Equal(storeadapter.ErrorKeyNotFound))

				desired, err := store.GetDesiredState()
				Ω(err).ShouldNot(HaveOccured())
				Ω(desired).Should(HaveLen(2))
				Ω(desired).Should(ContainElement(EqualDesiredState(app2.DesiredState(1))))
				Ω(desired).Should(ContainElement(EqualDesiredState(app3.DesiredState(1))))
			})
		})
	})
})
