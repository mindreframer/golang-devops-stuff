package storecassandra

import (
	"github.com/cloudfoundry/hm9000/models"
)

func (s *StoreCassandra) SavePendingStartMessages(startMessages ...models.PendingStartMessage) error {
	batch := s.newBatch()
	for _, startMessage := range startMessages {
		batch.Query(`INSERT INTO PendingStartMessages (app_guid, app_version, message_id, send_on, sent_on, keep_alive, index_to_start, priority, skip_verification, reason) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			startMessage.AppGuid,
			startMessage.AppVersion,
			startMessage.MessageId,
			startMessage.SendOn,
			startMessage.SentOn,
			startMessage.KeepAlive,
			startMessage.IndexToStart,
			startMessage.Priority,
			startMessage.SkipVerification,
			startMessage.StartReason)
	}

	return s.session.ExecuteBatch(batch)
}

func (s *StoreCassandra) GetPendingStartMessages() (map[string]models.PendingStartMessage, error) {
	startMessages := map[string]models.PendingStartMessage{}
	var err error

	iter := s.session.Query(`SELECT app_guid, app_version, message_id, send_on, sent_on, keep_alive, index_to_start, priority, skip_verification, reason FROM PendingStartMessages`).Iter()

	var messageId, appGuid, appVersion, reason string
	var sendOn, sentOn int64
	var keepAlive, indexToStart int
	var priority float64
	var skipVerification bool

	for iter.Scan(&appGuid, &appVersion, &messageId, &sendOn, &sentOn, &keepAlive, &indexToStart, &priority, &skipVerification, &reason) {
		startMessage := models.PendingStartMessage{
			PendingMessage: models.PendingMessage{
				MessageId:  messageId,
				SendOn:     sendOn,
				SentOn:     sentOn,
				KeepAlive:  keepAlive,
				AppGuid:    appGuid,
				AppVersion: appVersion,
			},
			IndexToStart:     indexToStart,
			Priority:         priority,
			SkipVerification: skipVerification,
			StartReason:      models.PendingStartMessageReason(reason),
		}
		startMessages[startMessage.StoreKey()] = startMessage
	}

	err = iter.Close()

	return startMessages, err
}

func (s *StoreCassandra) DeletePendingStartMessages(startMessages ...models.PendingStartMessage) error {
	batch := s.newBatch()
	for _, startMessage := range startMessages {
		batch.Query(`DELETE FROM PendingStartMessages WHERE app_guid=? AND app_version=? AND index_to_start=?`, startMessage.AppGuid, startMessage.AppVersion, startMessage.IndexToStart)
	}

	return s.session.ExecuteBatch(batch)
}
