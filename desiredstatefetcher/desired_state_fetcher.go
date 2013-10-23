package desiredstatefetcher

import (
	"fmt"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/httpclient"
	"github.com/cloudfoundry/hm9000/helpers/timeprovider"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/store"
	"io/ioutil"
	"net/http"
)

type DesiredStateFetcherResult struct {
	Success    bool
	Message    string
	Error      error
	NumResults int
}

const initialBulkToken = "{}"

type DesiredStateFetcher struct {
	config       config.Config
	httpClient   httpclient.HttpClient
	store        store.Store
	timeProvider timeprovider.TimeProvider
	cache        map[string]models.DesiredAppState
}

func New(config config.Config,
	store store.Store,
	httpClient httpclient.HttpClient,
	timeProvider timeprovider.TimeProvider) *DesiredStateFetcher {

	return &DesiredStateFetcher{
		config:       config,
		httpClient:   httpClient,
		store:        store,
		timeProvider: timeProvider,
		cache:        map[string]models.DesiredAppState{},
	}
}

func (fetcher *DesiredStateFetcher) Fetch(resultChan chan DesiredStateFetcherResult) {
	fetcher.cache = map[string]models.DesiredAppState{}

	authInfo := models.BasicAuthInfo{
		User:     fetcher.config.CCAuthUser,
		Password: fetcher.config.CCAuthPassword,
	}

	fetcher.fetchBatch(authInfo.Encode(), initialBulkToken, 0, resultChan)
}

func (fetcher *DesiredStateFetcher) fetchBatch(authorization string, token string, numResults int, resultChan chan DesiredStateFetcherResult) {
	req, err := http.NewRequest("GET", fetcher.bulkURL(fetcher.config.DesiredStateBatchSize, token), nil)

	if err != nil {
		resultChan <- DesiredStateFetcherResult{Message: "Failed to generate URL request", Error: err}
		return
	}

	req.Header.Add("Authorization", authorization)

	fetcher.httpClient.Do(req, func(resp *http.Response, err error) {
		if err != nil {
			resultChan <- DesiredStateFetcherResult{Message: "HTTP request failed with error", Error: err}
			return
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusUnauthorized {
			resultChan <- DesiredStateFetcherResult{Message: "HTTP request received unauthorized response code", Error: fmt.Errorf("Unauthorized")}
			return
		}

		if resp.StatusCode != http.StatusOK {
			resultChan <- DesiredStateFetcherResult{Message: fmt.Sprintf("HTTP request received non-200 response (%d)", resp.StatusCode), Error: fmt.Errorf("Invalid response code")}
			return
		}

		body, err := ioutil.ReadAll(resp.Body)

		if err != nil {
			resultChan <- DesiredStateFetcherResult{Message: "Failed to read HTTP response body", Error: err}
			return
		}

		response, err := NewDesiredStateServerResponse(body)
		if err != nil {
			resultChan <- DesiredStateFetcherResult{Message: "Failed to parse HTTP response body JSON", Error: err}
			return
		}

		if len(response.Results) == 0 {
			err = fetcher.syncStore()
			if err != nil {
				resultChan <- DesiredStateFetcherResult{Message: "Failed to sync desired state to the store", Error: err}
				return
			}

			fetcher.store.BumpDesiredFreshness(fetcher.timeProvider.Time())
			resultChan <- DesiredStateFetcherResult{Success: true, NumResults: numResults}
			return
		}

		fetcher.cacheResponse(response)
		fetcher.fetchBatch(authorization, response.BulkTokenRepresentation(), numResults+len(response.Results), resultChan)
	})
}

func (fetcher *DesiredStateFetcher) bulkURL(batchSize int, bulkToken string) string {
	return fmt.Sprintf("%s/bulk/apps?batch_size=%d&bulk_token=%s", fetcher.config.CCBaseURL, batchSize, bulkToken)
}

func (fetcher *DesiredStateFetcher) syncStore() error {
	desiredStates := make([]models.DesiredAppState, len(fetcher.cache))
	i := 0
	for _, desiredState := range fetcher.cache {
		desiredStates[i] = desiredState
		i++
	}
	err := fetcher.store.SaveDesiredState(desiredStates...)
	if err != nil {
		return err
	}

	storedDesiredState, err := fetcher.store.GetDesiredState()
	if err != nil {
		return err
	}

	statesToDelete := make([]models.DesiredAppState, 0)
	for _, desiredState := range storedDesiredState {
		_, present := fetcher.cache[desiredState.StoreKey()]
		if !present {
			statesToDelete = append(statesToDelete, desiredState)
		}
	}

	return fetcher.store.DeleteDesiredState(statesToDelete...)
}

func (fetcher *DesiredStateFetcher) cacheResponse(response DesiredStateServerResponse) {
	for _, desiredState := range response.Results {
		if desiredState.State == models.AppStateStarted {
			fetcher.cache[desiredState.StoreKey()] = desiredState
		}
	}
}
