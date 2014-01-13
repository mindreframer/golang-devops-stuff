package hm

import (
	"fmt"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/exiter"
	"github.com/cloudfoundry/hm9000/helpers/locker"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/helpers/metricsaccountant"
	"github.com/cloudfoundry/hm9000/helpers/timeprovider"
	"github.com/cloudfoundry/hm9000/helpers/workerpool"
	"github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelocker"
	"github.com/cloudfoundry/hm9000/testhelpers/faketimeprovider"
	"github.com/cloudfoundry/yagnats"
	"github.com/coreos/go-etcd/etcd"
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

func buildLocker(l logger.Logger, conf *config.Config, lockName string) locker.Locker {
	if conf.StoreType == "etcd" {
		etcdClient := etcd.NewClient(conf.StoreURLs)
		return locker.New(etcdClient, exiter.New(), lockName, 10)
	}

	l.Info("Looks like you're trying to use not-etcd with a locker...  lol!")
	return fakelocker.New()
}

func acquireLock(l logger.Logger, conf *config.Config, lockName string) {
	locker := buildLocker(l, conf, lockName)
	l.Info("Acquiring lock for " + lockName)
	err := locker.GetAndMaintainLock()
	if err != nil {
		l.Error("Failed to talk to lock store", err)
		os.Exit(1)
	}
	l.Info("Acquired lock for " + lockName)
}

func connectToStoreAdapter(l logger.Logger, conf *config.Config) (storeadapter.StoreAdapter, metricsaccountant.UsageTracker) {
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

func connectToStore(l logger.Logger, conf *config.Config) (store.Store, metricsaccountant.UsageTracker) {
	if conf.StoreType == "etcd" || conf.StoreType == "ZooKeeper" {
		adapter, workerPool := connectToStoreAdapter(l, conf)
		return store.NewStore(conf, adapter, l), workerPool
	} else {
		l.Error(fmt.Sprintf("Unknown store type %s.  Choose one of 'etcd' or 'ZooKeeper'", conf.StoreType), fmt.Errorf("Unkown store type"))
		os.Exit(1)
	}

	return nil, nil
}
