package hm

import (
	"fmt"
	"github.com/cloudfoundry/go_cfmessagebus"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/helpers/metricsaccountant"
	"github.com/cloudfoundry/hm9000/helpers/timeprovider"
	"github.com/cloudfoundry/hm9000/helpers/workerpool"
	"github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"github.com/cloudfoundry/hm9000/storecassandra"
	"github.com/cloudfoundry/hm9000/testhelpers/faketimeprovider"
	"github.com/cloudfoundry/yagnats"
	"strconv"
	"time"

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

func connectToMessageBus(l logger.Logger, conf config.Config) yagnats.NATSClient {
	connectionInfo := &yagnats.ConnectionInfo{
		Addr: fmt.Sprintf("%s:%d", conf.NATS.Host, conf.NATS.Port),

		Username: conf.NATS.User,
		Password: conf.NATS.Password,
	}

	natsClient := yagnats.NewClient()

	err := natsClient.Connect(connectionInfo)

	if err != nil {
		l.Error("Failed to connect to the message bus", err)
		os.Exit(1)
	}

	return natsClient
}

func connectToCFMessageBus(l logger.Logger, conf config.Config) cfmessagebus.MessageBus {
	messageBus, err := cfmessagebus.NewMessageBus("NATS")
	if err != nil {
		l.Error("Failed to initialize the CF message bus", err)
		os.Exit(1)
	}

	messageBus.Configure(conf.NATS.Host, conf.NATS.Port, conf.NATS.User, conf.NATS.Password)
	err = messageBus.Connect()

	if err != nil {
		l.Error("Failed to connect to the CF message bus", err)
		os.Exit(1)
	}

	return messageBus
}

func connectToStoreAdapter(l logger.Logger, conf config.Config) (storeadapter.StoreAdapter, metricsaccountant.UsageTracker) {
	var adapter storeadapter.StoreAdapter
	workerPool := workerpool.NewWorkerPool(conf.StoreMaxConcurrentRequests)
	if conf.StoreType == "etcd" {
		adapter = storeadapter.NewETCDStoreAdapter(conf.StoreURLs, workerPool)
	} else if conf.StoreType == "ZooKeeper" {
		adapter = storeadapter.NewZookeeperStoreAdapter(conf.StoreURLs, workerPool, buildTimeProvider(l), time.Second)
	} else {
		l.Error(fmt.Sprintf("Unknown store type %s.  Choose one of 'etcd' or 'ZooKeeper'", conf.StoreType), fmt.Errorf("Unkown store type"))
		os.Exit(1)
	}
	err := adapter.Connect()
	if err != nil {
		l.Error("Failed to connect to the store", err)
		os.Exit(1)
	}

	return adapter, workerPool
}

func connectToStore(l logger.Logger, conf config.Config) (store.Store, metricsaccountant.UsageTracker) {
	if conf.StoreType == "etcd" || conf.StoreType == "ZooKeeper" {
		adapter, workerPool := connectToStoreAdapter(l, conf)
		return store.NewStore(conf, adapter, l), workerPool
	} else if conf.StoreType == "Cassandra" {
		store, err := storecassandra.New(conf.StoreURLs, conf.CassandraConsistency(), conf, buildTimeProvider(l))
		if err != nil {
			l.Error("Failed to connect to the store", err)
			os.Exit(1)
		}
		return store, nil
	} else {
		l.Error(fmt.Sprintf("Unknown store type %s.  Choose one of 'etcd', 'ZooKeeper' or 'Cassandra'", conf.StoreType), fmt.Errorf("Unkown store type"))
		os.Exit(1)
	}

	return nil, nil
}
