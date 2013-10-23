package actualstatelistener

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/helpers/timeprovider"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/store"

	"github.com/cloudfoundry/yagnats"
)

type ActualStateListener struct {
	logger       logger.Logger
	config       config.Config
	messageBus   yagnats.NATSClient
	store        store.Store
	timeProvider timeprovider.TimeProvider
}

func New(config config.Config,
	messageBus yagnats.NATSClient,
	store store.Store,
	timeProvider timeprovider.TimeProvider,
	logger logger.Logger) *ActualStateListener {

	return &ActualStateListener{
		logger:       logger,
		config:       config,
		messageBus:   messageBus,
		store:        store,
		timeProvider: timeProvider,
	}
}

func (listener *ActualStateListener) Start() {
	listener.messageBus.Subscribe("dea.advertise", func(message *yagnats.Message) {
		listener.logger.Info("Received dea.advertise, bumping freshness.")
		listener.bumpFreshness()
	})

	listener.messageBus.Subscribe("dea.heartbeat", func(message *yagnats.Message) {
		heartbeat, err := models.NewHeartbeatFromJSON([]byte(message.Payload))
		if err != nil {
			listener.logger.Error("Could not unmarshal heartbeat", err,
				map[string]string{
					"MessageBody": message.Payload,
				})
			return
		}

		err = listener.store.SaveActualState(heartbeat.InstanceHeartbeats...)
		if err != nil {
			listener.logger.Error("Could not put instance heartbeats in store:", err)
			return
		}

		listener.logger.Info("Received dea.heartbeat, bumping freshness.")
		listener.bumpFreshness()
	})
}

func (listener *ActualStateListener) bumpFreshness() {
	err := listener.store.BumpActualFreshness(listener.timeProvider.Time())
	if err != nil {
		listener.logger.Error("Could not update actual freshness", err)
	}
}
