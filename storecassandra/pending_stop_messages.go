package storecassandra

import (
	"github.com/cloudfoundry/hm9000/models"
)

func (s *StoreCassandra) SavePendingStopMessages(stopMessages ...models.PendingStopMessage) error {
	batch := s.newBatch()
	for _, stopMessage := range stopMessages {
		batch.Query(`INSERT INTO PendingStopMessages (app_guid, app_version, message_id, send_on, sent_on, keep_alive, instance_guid, reason) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			stopMessage.AppGuid,
			stopMessage.AppVersion,
			stopMessage.MessageId,
			stopMessage.SendOn,
			stopMessage.SentOn,
			stopMessage.KeepAlive,
			stopMessage.InstanceGuid,
			stopMessage.StopReason)

	}

	return s.session.ExecuteBatch(batch)
}

func (s *StoreCassandra) GetPendingStopMessages() (map[string]models.PendingStopMessage, error) {
	stopMessages := map[string]models.PendingStopMessage{}
	var err error

	iter := s.session.Query(`SELECT app_guid, app_version, message_id, send_on, sent_on, keep_alive, instance_guid, reason FROM PendingStopMessages`).Iter()

	var messageId, appGuid, appVersion, instanceGuid, reason string
	var sendOn, sentOn int64
	var keepAlive int

	for iter.Scan(&appGuid, &appVersion, &messageId, &sendOn, &sentOn, &keepAlive, &instanceGuid, &reason) {
		stopMessage := models.PendingStopMessage{
			PendingMessage: models.PendingMessage{
				MessageId:  messageId,
				SendOn:     sendOn,
				SentOn:     sentOn,
				KeepAlive:  keepAlive,
				AppGuid:    appGuid,
				AppVersion: appVersion,
			},
			InstanceGuid: instanceGuid,
			StopReason:   models.PendingStopMessageReason(reason),
		}
		stopMessages[stopMessage.StoreKey()] = stopMessage
	}

	err = iter.Close()

	return stopMessages, err
}

func (s *StoreCassandra) DeletePendingStopMessages(stopMessages ...models.PendingStopMessage) error {
	batch := s.newBatch()
	for _, stopMessage := range stopMessages {
		batch.Query(`DELETE FROM PendingStopMessages WHERE app_guid=? AND app_version=? AND instance_guid=?`,
			stopMessage.AppGuid,
			stopMessage.AppVersion,
			stopMessage.InstanceGuid)
	}

	return s.session.ExecuteBatch(batch)
}
