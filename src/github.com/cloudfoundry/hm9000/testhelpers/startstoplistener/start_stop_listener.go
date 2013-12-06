package startstoplistener

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/yagnats"
)

type StartStopListener struct {
	Starts     []models.StartMessage
	Stops      []models.StopMessage
	messageBus yagnats.NATSClient
}

func NewStartStopListener(messageBus yagnats.NATSClient, conf *config.Config) *StartStopListener {
	listener := &StartStopListener{
		messageBus: messageBus,
	}

	messageBus.Subscribe(conf.SenderNatsStartSubject, func(message *yagnats.Message) {
		startMessage, err := models.NewStartMessageFromJSON([]byte(message.Payload))
		if err != nil {
			panic(err)
		}
		listener.Starts = append(listener.Starts, startMessage)
	})

	messageBus.Subscribe(conf.SenderNatsStopSubject, func(message *yagnats.Message) {
		stopMessage, err := models.NewStopMessageFromJSON([]byte(message.Payload))
		if err != nil {
			panic(err)
		}
		listener.Stops = append(listener.Stops, stopMessage)
	})

	return listener
}

func (listener *StartStopListener) Reset() {
	listener.Starts = make([]models.StartMessage, 0)
	listener.Stops = make([]models.StopMessage, 0)
}
