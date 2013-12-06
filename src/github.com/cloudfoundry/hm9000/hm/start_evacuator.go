package hm

import (
	"github.com/cloudfoundry/hm9000/config"
	evacuatorpackage "github.com/cloudfoundry/hm9000/evacuator"
	"github.com/cloudfoundry/hm9000/helpers/logger"
)

func StartEvacuator(l logger.Logger, conf *config.Config) {
	messageBus := connectToMessageBus(l, conf)
	store, _ := connectToStore(l, conf)

	evacuator := evacuatorpackage.New(messageBus, store, buildTimeProvider(l), l)

	evacuator.Listen()
	l.Info("Listening for DEA Evacuations")
	select {}
}
