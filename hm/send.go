package hm

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/sender"
	"github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/storeadapter"
	"github.com/cloudfoundry/yagnats"

	"os"
)

func Send(l logger.Logger, conf config.Config, poll bool) {
	messageBus := connectToMessageBus(l, conf)
	etcdStoreAdapter := connectToETCDStoreAdapter(l, conf)

	if poll {
		l.Info("Starting Sender Daemon...")
		err := Daemonize("Sender", func() error {
			return send(l, conf, messageBus, etcdStoreAdapter)
		}, conf.SenderPollingInterval(), conf.SenderTimeout(), l)
		if err != nil {
			l.Error("Sender Daemon Errored", err)
		}
		l.Info("Sender Daemon is Down")
	} else {
		err := send(l, conf, messageBus, etcdStoreAdapter)
		if err != nil {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}
}

func send(l logger.Logger, conf config.Config, messageBus yagnats.NATSClient, etcdStoreAdapter storeadapter.StoreAdapter) error {
	store := store.NewStore(conf, etcdStoreAdapter, l)
	l.Info("Sending...")

	sender := sender.New(store, conf, messageBus, buildTimeProvider(l), l)
	err := sender.Send()

	if err != nil {
		l.Error("Sender failed with error", err)
		return err
	} else {
		l.Info("Sender completed succesfully")
		return nil
	}
}
