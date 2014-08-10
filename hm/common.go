package hm

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/cloudfoundry/gunk/timeprovider"
	"github.com/cloudfoundry/gunk/timeprovider/faketimeprovider"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/helpers/metricsaccountant"
	"github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/storeadapter"
	"github.com/cloudfoundry/storeadapter/etcdstoreadapter"
	"github.com/cloudfoundry/storeadapter/workerpool"
	"github.com/cloudfoundry/yagnats"

	"os"
)

func buildTimeProvider(l logger.Logger) timeprovider.TimeProvider {
	if os.Getenv("HM9000_FAKE_TIME") == "" {
		return timeprovider.NewTimeProvider()
	} else {
		timestamp, err := strconv.Atoi(os.Getenv("HM9000_FAKE_TIME"))
		if err != nil {
			l.Error("Failed to load timestamp", err)
			os.Exit(1)
		}
		return &faketimeprovider.FakeTimeProvider{
			TimeToProvide: time.Unix(int64(timestamp), 0),
		}
	}
}

func connectToMessageBus(l logger.Logger, conf *config.Config) yagnats.NATSClient {
	members := []yagnats.ConnectionProvider{}

	for _, natsConf := range conf.NATS {
		members = append(members, &yagnats.ConnectionInfo{
			Addr: fmt.Sprintf("%s:%d", natsConf.Host, natsConf.Port),

			Username: natsConf.User,
			Password: natsConf.Password,
		})
	}

	connectionInfo := &yagnats.ConnectionCluster{
		Members: members,
	}

	natsClient := yagnats.NewClient()

	err := natsClient.Connect(connectionInfo)

	if err != nil {
		l.Error("Failed to connect to the message bus", err)
		os.Exit(1)
	}

	return natsClient
}

func acquireLock(l logger.Logger, conf *config.Config, lockName string) {
	adapter, _ := connectToStoreAdapter(l, conf)
	l.Info("Acquiring lock for " + lockName)

	lock := storeadapter.StoreNode{
		Key: "/hm/locks/" + lockName,
		TTL: 10,
	}

	status, _, err := adapter.MaintainNode(lock)
	if err != nil {
		l.Error("Failed to talk to lock store", err)
		os.Exit(1)
	}

	lockAcquired := make(chan bool)

	go func() {
		for {
			if <-status {
				if lockAcquired != nil {
					close(lockAcquired)
					lockAcquired = nil
				}
			} else {
				l.Error("Lost the lock", errors.New("Lost the lock"))
				os.Exit(197)
			}
		}
	}()

	<-lockAcquired
	l.Info("Acquired lock for " + lockName)
}

func connectToStoreAdapter(l logger.Logger, conf *config.Config) (storeadapter.StoreAdapter, metricsaccountant.UsageTracker) {
	var adapter storeadapter.StoreAdapter
	workerPool := workerpool.NewWorkerPool(conf.StoreMaxConcurrentRequests)
	adapter = etcdstoreadapter.NewETCDStoreAdapter(conf.StoreURLs, workerPool)
	err := adapter.Connect()
	if err != nil {
		l.Error("Failed to connect to the store", err)
		os.Exit(1)
	}

	return adapter, workerPool
}

func connectToStore(l logger.Logger, conf *config.Config) (store.Store, metricsaccountant.UsageTracker) {
	adapter, workerPool := connectToStoreAdapter(l, conf)
	return store.NewStore(conf, adapter, l), workerPool
}
