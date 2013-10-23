package models

import (
	"encoding/json"
	"strconv"
)

type InstanceState string

const (
	InstanceStateInvalid  InstanceState = ""
	InstanceStateStarting InstanceState = "STARTING"
	InstanceStateRunning  InstanceState = "RUNNING"
	InstanceStateCrashed  InstanceState = "CRASHED"
)

type InstanceHeartbeat struct {
	CCPartition    string        `json:"cc_partition"`
	AppGuid        string        `json:"droplet"`
	AppVersion     string        `json:"version"`
	InstanceGuid   string        `json:"instance"`
	InstanceIndex  int           `json:"index"`
	State          InstanceState `json:"state"`
	StateTimestamp float64       `json:"state_timestamp"`
}

func NewInstanceHeartbeatFromJSON(encoded []byte) (InstanceHeartbeat, error) {
	var instance InstanceHeartbeat
	err := json.Unmarshal(encoded, &instance)
	if err != nil {
		return InstanceHeartbeat{}, err
	}
	return instance, nil
}

func (instance InstanceHeartbeat) ToJSON() []byte {
	encoded, _ := json.Marshal(instance)
	return encoded
}

func (instance InstanceHeartbeat) StoreKey() string {
	return instance.InstanceGuid
}

func (instance InstanceHeartbeat) IsStartingOrRunning() bool {
	return instance.IsStarting() || instance.IsRunning()
}

func (instance InstanceHeartbeat) IsStarting() bool {
	return instance.State == InstanceStateStarting
}

func (instance InstanceHeartbeat) IsRunning() bool {
	return instance.State == InstanceStateRunning
}

func (instance InstanceHeartbeat) IsCrashed() bool {
	return instance.State == InstanceStateCrashed
}

func (instance InstanceHeartbeat) LogDescription() map[string]string {
	return map[string]string{
		"AppGuid":        instance.AppGuid,
		"AppVersion":     instance.AppVersion,
		"InstanceGuid":   instance.InstanceGuid,
		"InstanceIndex":  strconv.Itoa(instance.InstanceIndex),
		"State":          string(instance.State),
		"StateTimestamp": strconv.Itoa(int(instance.StateTimestamp)),
	}
}

type Heartbeat struct {
	DeaGuid            string              `json:"dea"`
	InstanceHeartbeats []InstanceHeartbeat `json:"droplets"`
}

func NewHeartbeatFromJSON(encoded []byte) (Heartbeat, error) {
	var heartbeat Heartbeat
	err := json.Unmarshal(encoded, &heartbeat)
	if err != nil {
		return Heartbeat{}, err
	}
	return heartbeat, nil
}

func (heartbeat Heartbeat) ToJSON() []byte {
	encoded, _ := json.Marshal(heartbeat)
	return encoded
}
