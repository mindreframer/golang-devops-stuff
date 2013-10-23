package hm

import (
	"github.com/cloudfoundry/hm9000/actualstatelistener"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
)

func StartListeningForActual(l logger.Logger, conf config.Config) {
	messageBus := connectToMessageBus(l, conf)
	store := connectToStore(l, conf)

	listener := actualstatelistener.New(conf,
		messageBus,
		store,
		buildTimeProvider(l),
		l)

	listener.Start()
	l.Info("Listening for Actual State")
	select {}
}
