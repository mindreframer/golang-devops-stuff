package md_test

import (
	"github.com/cloudfoundry/hm9000/desiredstatefetcher"
	"github.com/cloudfoundry/hm9000/helpers/httpclient"
	"github.com/cloudfoundry/hm9000/helpers/timeprovider"
	"github.com/cloudfoundry/hm9000/models"
	storepackage "github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/testhelpers/appfixture"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Fetching from CC and storing the result in the Store", func() {
	var (
		fetcher    *desiredstatefetcher.DesiredStateFetcher
		a1         appfixture.AppFixture
		a2         appfixture.AppFixture
		a3         appfixture.AppFixture
		store      storepackage.Store
		resultChan chan desiredstatefetcher.DesiredStateFetcherResult
	)

	BeforeEach(func() {
		resultChan = make(chan desiredstatefetcher.DesiredStateFetcherResult, 1)
		a1 = appfixture.NewAppFixture()
		a2 = appfixture.NewAppFixture()
		a3 = appfixture.NewAppFixture()

		stateServer.SetDesiredState([]models.DesiredAppState{
			a1.DesiredState(1),
			a2.DesiredState(1),
			a3.DesiredState(1),
		})

		conf.CCBaseURL = desiredStateServerBaseUrl

		store = storepackage.NewStore(conf, storeAdapter, fakelogger.NewFakeLogger())

		fetcher = desiredstatefetcher.New(conf, store, httpclient.NewHttpClient(conf.FetcherNetworkTimeout()), &timeprovider.RealTimeProvider{})
		fetcher.Fetch(resultChan)
	})

	It("requests for the first set of data from the CC and stores the response", func() {
		var desired map[string]models.DesiredAppState
		var err error
		Eventually(func() interface{} {
			desired, err = store.GetDesiredState()
			return desired
		}, 1, 0.1).ShouldNot(BeEmpty())

		Ω(desired).Should(HaveKey(a1.AppGuid + "-" + a1.AppVersion))
		Ω(desired).Should(HaveKey(a2.AppGuid + "-" + a2.AppVersion))
		Ω(desired).Should(HaveKey(a3.AppGuid + "-" + a3.AppVersion))
	})

	It("bumps the freshness", func() {
		Eventually(func() error {
			_, err := storeAdapter.Get(conf.DesiredFreshnessKey)
			return err
		}, 1, 0.1).ShouldNot(HaveOccured())
	})

	It("reports success to the channel", func() {
		result := <-resultChan
		Ω(result.Success).Should(BeTrue())
		Ω(result.NumResults).Should(Equal(3))
		Ω(result.Message).Should(BeZero())
		Ω(result.Error).ShouldNot(HaveOccured())
	})

	Context("when fetching again, and apps have been stopped and/or deleted", func() {
		BeforeEach(func() {
			<-resultChan

			desired1 := a1.DesiredState(1)
			desired1.State = models.AppStateStopped

			stateServer.SetDesiredState([]models.DesiredAppState{
				desired1,
				a3.DesiredState(1),
			})

			fetcher.Fetch(resultChan)
		})

		It("should remove those apps from the store", func() {
			<-resultChan

			desired, err := store.GetDesiredState()
			Ω(err).ShouldNot(HaveOccured())
			Ω(desired).Should(HaveLen(1))
			Ω(desired).Should(HaveKey(a3.AppGuid + "-" + a3.AppVersion))
		})
	})
})
