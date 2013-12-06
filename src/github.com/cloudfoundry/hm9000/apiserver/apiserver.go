package apiserver

import (
	"encoding/json"
	"fmt"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/helpers/timeprovider"
	"github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/yagnats"
	"time"
)

type ApiServer struct {
	messageBus   yagnats.NATSClient
	store        store.Store
	timeProvider timeprovider.TimeProvider
	logger       logger.Logger
}

type AppStateRequest struct {
	AppGuid    string `json:"droplet"`
	AppVersion string `json:"version"`
}

func New(messageBus yagnats.NATSClient, store store.Store, timeProvider timeprovider.TimeProvider, logger logger.Logger) *ApiServer {
	return &ApiServer{
		messageBus:   messageBus,
		store:        store,
		timeProvider: timeProvider,
		logger:       logger,
	}
}

func (server *ApiServer) Listen() {
	server.messageBus.SubscribeWithQueue("app.state", "hm9000", func(message *yagnats.Message) {
		if message.ReplyTo == "" {
			return
		}

		t := time.Now()

		var err error
		var response []byte

		defer func() {
			if err != nil {
				server.messageBus.Publish(message.ReplyTo, []byte("{}"))
				server.logger.Error("Failed to handle app.state request", err, map[string]string{
					"payload":      string(message.Payload),
					"elapsed time": fmt.Sprintf("%s", time.Since(t)),
				})
				return
			} else {
				server.messageBus.Publish(message.ReplyTo, response)
				server.logger.Info("Responded succesfully to app.state request", map[string]string{
					"payload":      string(message.Payload),
					"elapsed time": fmt.Sprintf("%s", time.Since(t)),
				})
			}
		}()

		var request AppStateRequest
		err = json.Unmarshal([]byte(message.Payload), &request)
		if err != nil {
			return
		}

		err = server.store.VerifyFreshness(server.timeProvider.Time())
		if err != nil {
			return
		}

		app, err := server.store.GetApp(request.AppGuid, request.AppVersion)
		if err != nil {
			return
		}

		response = app.ToJSON()
		return
	})
}
