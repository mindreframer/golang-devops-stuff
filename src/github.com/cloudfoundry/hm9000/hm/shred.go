package hm

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/shredder"
	"github.com/cloudfoundry/hm9000/store"
	"os"
)

func Shred(l logger.Logger, conf config.Config, poll bool) {
	store, _ := connectToStore(l, conf)

	if poll {
		l.Info("Starting Shredder Daemon...")
		err := Daemonize("Shredder", func() error {
			return shred(l, store)
		}, conf.ShredderPollingInterval(), conf.ShredderTimeout(), l)
		if err != nil {
			l.Error("Shredder Errored", err)
		}
		l.Info("Shredder Daemon is Down")
		os.Exit(1)
	} else {
		err := shred(l, store)
		if err != nil {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
	}
}

func shred(l logger.Logger, store store.Store) error {
	l.Info("Shredding Store")
	theShredder := shredder.New(store)
	return theShredder.Shred()
}
