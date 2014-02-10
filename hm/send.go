package hm

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/helpers/metricsaccountant"
	"github.com/cloudfoundry/hm9000/sender"
	"github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/yagnats"

	"os"
)

func Send(l logger.Logger, conf *config.Config, poll bool) {
	messageBus := connectToMessageBus(l, conf)
	store, _ := connectToStore(l, conf)

	if poll {
		l.Info("Starting Sender Daemon...")

		adapter, _ := connectToStoreAdapter(l, conf)

		err := Daemonize("Sender", func() error {
			return send(l, conf, messageBus, store)
		}, conf.SenderPollingInterval(), conf.SenderTimeout(), l, adapter)
		if err != nil {
			l.Error("Sender Daemon Errored", err)
		}
		l.Info("Sender Daemon is Down")
		os.Exit(1)
	} else {
		err := send(l, conf, messageBus, store)
		if err != nil {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}
}

func send(l logger.Logger, conf *config.Config, messageBus yagnats.NATSClient, store store.Store) error {
	l.Info("Sending...")

	sender := sender.New(store, metricsaccountant.New(store), conf, messageBus, buildTimeProvider(l), l)
	err := sender.Send()

	if err != nil {
		l.Error("Sender failed with error", err)
		return err
	} else {
		l.Info("Sender completed succesfully")
		return nil
	}
}
