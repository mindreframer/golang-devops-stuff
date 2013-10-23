package models

import (
	"encoding/json"
)

type StartMessage struct {
	MessageId     string `json:"message_id"`
	AppGuid       string `json:"droplet"`
	AppVersion    string `json:"version"`
	InstanceIndex int    `json:"instance_index"`
}

type StopMessage struct {
	MessageId     string `json:"message_id"`
	AppGuid       string `json:"droplet"`
	AppVersion    string `json:"version"`
	InstanceGuid  string `json:"instance_guid"`
	InstanceIndex int    `json:"instance_index"`
	IsDuplicate   bool   `json:"is_duplicate"`
}

func NewStartMessageFromJSON(encoded []byte) (StartMessage, error) {
	message := StartMessage{}
	err := json.Unmarshal(encoded, &message)
	if err != nil {
		return StartMessage{}, err
	}
	return message, nil
}

func NewStopMessageFromJSON(encoded []byte) (StopMessage, error) {
	message := StopMessage{}
	err := json.Unmarshal(encoded, &message)
	if err != nil {
		return StopMessage{}, err
	}
	return message, nil
}

func (message StartMessage) ToJSON() []byte {
	result, _ := json.Marshal(message)
	return result
}

func (message StopMessage) ToJSON() []byte {
	result, _ := json.Marshal(message)
	return result
}
