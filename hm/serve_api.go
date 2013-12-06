package hm

import (
	"fmt"
	"github.com/cloudfoundry/hm9000/apiserver"
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
)

func ServeAPI(l logger.Logger, conf *config.Config) {
	store, _ := connectToStore(l, conf)
	messageBus := connectToMessageBus(l, conf)

	apiServer := apiserver.New(
		messageBus,
		store,
		buildTimeProvider(l),
		l,
	)

	apiServer.Listen()
	l.Info(fmt.Sprintf("Serving API over NATS (subject: app.state)"))
	select {}
}
