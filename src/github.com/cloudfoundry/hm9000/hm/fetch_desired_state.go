package hm

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/desiredstatefetcher"
	"github.com/cloudfoundry/hm9000/helpers/httpclient"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/helpers/metricsaccountant"
	"github.com/cloudfoundry/hm9000/store"
	"os"
	"strconv"
)

func FetchDesiredState(l logger.Logger, conf *config.Config, poll bool) {
	store, _ := connectToStore(l, conf)

	if poll {
		l.Info("Starting Desired State Daemon...")
		err := Daemonize("Fetcher", func() error {
			return fetchDesiredState(l, conf, store)
		}, conf.FetcherPollingInterval(), conf.FetcherTimeout(), l)
		if err != nil {
			l.Error("Desired State Daemon Errored", err)
		}
		l.Info("Desired State Daemon is Down")
		os.Exit(1)
	} else {
		err := fetchDesiredState(l, conf, store)
		if err != nil {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}
}

func fetchDesiredState(l logger.Logger, conf *config.Config, store store.Store) error {
	l.Info("Fetching Desired State")
	fetcher := desiredstatefetcher.New(conf,
		store,
		metricsaccountant.New(store),
		httpclient.NewHttpClient(conf.FetcherNetworkTimeout()),
		buildTimeProvider(l),
		l,
	)

	resultChan := make(chan desiredstatefetcher.DesiredStateFetcherResult, 1)
	fetcher.Fetch(resultChan)

	result := <-resultChan

	if result.Success {
		l.Info("Success", map[string]string{"Number of Desired Apps Fetched": strconv.Itoa(result.NumResults)})
		return nil
	} else {
		l.Error(result.Message, result.Error)
		return result.Error
	}
	return nil
}
