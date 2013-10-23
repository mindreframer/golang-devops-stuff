package hm

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/desiredstatefetcher"
	"github.com/cloudfoundry/hm9000/helpers/httpclient"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"os"
	"strconv"
)

func FetchDesiredState(l logger.Logger, conf config.Config, poll bool) {
	etcdStoreAdapter := connectToETCDStoreAdapter(l, conf)

	if poll {
		l.Info("Starting Desired State Daemon...")
		err := Daemonize("Fetcher", func() error {
			return fetchDesiredState(l, conf, etcdStoreAdapter)
		}, conf.FetcherPollingInterval(), conf.FetcherTimeout(), l)
		if err != nil {
			l.Error("Desired State Daemon Errored", err)
		}
		l.Info("Desired State Daemon is Down")
	} else {
		err := fetchDesiredState(l, conf, etcdStoreAdapter)
		if err != nil {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}
}

func fetchDesiredState(l logger.Logger, conf config.Config, etcdStoreAdapter storeadapter.StoreAdapter) error {
	l.Info("Fetching Desired State")
	store := store.NewStore(conf, etcdStoreAdapter, l)

	fetcher := desiredstatefetcher.New(conf,
		store,
		httpclient.NewHttpClient(conf.FetcherNetworkTimeout()),
		buildTimeProvider(l),
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
