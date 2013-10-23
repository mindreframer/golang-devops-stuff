package models

import (
	"encoding/json"
	"strconv"
	"time"
)

type PendingMessage struct {
	MessageId  string `json:"message_id"`
	SendOn     int64  `json:"send_on"`
	SentOn     int64  `json:"sent_on"`
	KeepAlive  int    `json:"keep_alive"`
	AppGuid    string `json:"droplet"`
	AppVersion string `json:"version"`
}

type PendingStartMessage struct {
	PendingMessage
	IndexToStart int     `json:"index"`
	Priority     float64 `json:"priority"`
}

type PendingStopMessage struct {
	PendingMessage
	InstanceGuid string `json:"instance"`
}

func newPendingMessage(now time.Time, delayInSeconds int, keepAliveInSeconds int, appGuid string, appVersion string) PendingMessage {
	return PendingMessage{
		SendOn:     now.Add(time.Duration(delayInSeconds) * time.Second).Unix(),
		SentOn:     0,
		KeepAlive:  keepAliveInSeconds,
		AppGuid:    appGuid,
		AppVersion: appVersion,
		MessageId:  Guid(),
	}
}

func (message PendingMessage) pendingLogDescription() map[string]string {
	return map[string]string{
		"SendOn":     time.Unix(message.SendOn, 0).String(),
		"SentOn":     time.Unix(message.SentOn, 0).String(),
		"KeepAlive":  strconv.Itoa(int(message.KeepAlive)),
		"MessageId":  message.MessageId,
		"AppGuid":    message.AppGuid,
		"AppVersion": message.AppVersion,
	}
}

func (message PendingMessage) pendingEqual(another PendingMessage) bool {
	return message.SendOn == another.SendOn &&
		message.SentOn == another.SentOn &&
		message.KeepAlive == another.KeepAlive &&
		message.AppGuid == another.AppGuid &&
		message.AppVersion == another.AppVersion
}

func (message PendingMessage) HasBeenSent() bool {
	return message.SentOn != 0
}

func (message PendingMessage) IsTimeToSend(currentTime time.Time) bool {
	return !message.HasBeenSent() && message.SendOn <= currentTime.Unix()
}

func (message PendingMessage) IsExpired(currentTime time.Time) bool {
	return message.HasBeenSent() && message.SentOn+int64(message.KeepAlive) <= currentTime.Unix()
}

func NewPendingStartMessage(now time.Time, delayInSeconds int, keepAliveInSeconds int, appGuid string, appVersion string, indexToStart int, priority float64) PendingStartMessage {
	return PendingStartMessage{
		PendingMessage: newPendingMessage(now, delayInSeconds, keepAliveInSeconds, appGuid, appVersion),
		IndexToStart:   indexToStart,
		Priority:       priority,
	}
}

func NewPendingStartMessageFromJSON(encoded []byte) (PendingStartMessage, error) {
	message := PendingStartMessage{}
	err := json.Unmarshal(encoded, &message)
	if err != nil {
		return PendingStartMessage{}, err
	}
	return message, nil
}

func (message PendingStartMessage) StoreKey() string {
	return message.AppGuid + "-" + message.AppVersion + "-" + strconv.Itoa(message.IndexToStart)
}

func (message PendingStartMessage) ToJSON() []byte {
	encoded, _ := json.Marshal(message)
	return encoded
}

func (message PendingStartMessage) LogDescription() map[string]string {
	base := message.pendingLogDescription()
	base["IndexToStart"] = strconv.Itoa(message.IndexToStart)
	return base
}

func (message PendingStartMessage) Equal(another PendingStartMessage) bool {
	return message.pendingEqual(another.PendingMessage) &&
		message.IndexToStart == another.IndexToStart &&
		message.Priority == another.Priority
}

func NewPendingStopMessage(now time.Time, delayInSeconds int, keepAliveInSeconds int, appGuid string, appVersion string, instanceGuid string) PendingStopMessage {
	return PendingStopMessage{
		PendingMessage: newPendingMessage(now, delayInSeconds, keepAliveInSeconds, appGuid, appVersion),
		InstanceGuid:   instanceGuid,
	}
}

func NewPendingStopMessageFromJSON(encoded []byte) (PendingStopMessage, error) {
	message := PendingStopMessage{}
	err := json.Unmarshal(encoded, &message)
	if err != nil {
		return PendingStopMessage{}, err
	}
	return message, nil
}

func (message PendingStopMessage) ToJSON() []byte {
	encoded, _ := json.Marshal(message)
	return encoded
}

func (message PendingStopMessage) StoreKey() string {
	return message.InstanceGuid
}

func (message PendingStopMessage) LogDescription() map[string]string {
	base := message.pendingLogDescription()
	base["InstanceGuid"] = message.InstanceGuid
	return base
}

func (message PendingStopMessage) Equal(another PendingStopMessage) bool {
	return message.pendingEqual(another.PendingMessage) &&
		message.InstanceGuid == another.InstanceGuid
}
