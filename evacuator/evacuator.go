package evacuator

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/helpers/timeprovider"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/yagnats"
)

type Evacuator struct {
	messageBus   yagnats.NATSClient
	store        store.Store
	timeProvider timeprovider.TimeProvider
	config       *config.Config
	logger       logger.Logger
}

func New(messageBus yagnats.NATSClient, store store.Store, timeProvider timeprovider.TimeProvider, config *config.Config, logger logger.Logger) *Evacuator {
	return &Evacuator{
		messageBus:   messageBus,
		store:        store,
		timeProvider: timeProvider,
		config:       config,
		logger:       logger,
	}
}

func (e *Evacuator) Listen() {
	e.messageBus.Subscribe("droplet.exited", func(message *yagnats.Message) {
		dropletExited, err := models.NewDropletExitedFromJSON([]byte(message.Payload))
		if err != nil {
			e.logger.Error("Failed to parse droplet exited message", err)
			return
		}

		e.handleExited(dropletExited)
	})
}

func (e *Evacuator) handleExited(exited models.DropletExited) {
	switch exited.Reason {
	case models.DropletExitedReasonDEAShutdown, models.DropletExitedReasonDEAEvacuation:
		startMessage := models.NewPendingStartMessage(
			e.timeProvider.Time(),
			0,
			e.config.GracePeriod(),
			exited.AppGuid,
			exited.AppVersion,
			exited.InstanceIndex,
			2.0,
			models.PendingStartMessageReasonEvacuating,
		)
		startMessage.SkipVerification = true

		e.logger.Info("Scheduling start message for droplet.exited message", startMessage.LogDescription(), exited.LogDescription())

		e.store.SavePendingStartMessages(startMessage)
	}
}
