package models

import (
	"encoding/json"
	"strconv"
)

type DropletExitedReason string

const (
	DropletExitedReasonInvalid       DropletExitedReason = ""
	DropletExitedReasonStopped       DropletExitedReason = "STOPPED"
	DropletExitedReasonCrashed       DropletExitedReason = "CRASHED"
	DropletExitedReasonDEAShutdown   DropletExitedReason = "DEA_SHUTDOWN"
	DropletExitedReasonDEAEvacuation DropletExitedReason = "DEA_EVACUATION"
)

type DropletExited struct {
	CCPartition     string              `json:"cc_partition"`
	AppGuid         string              `json:"droplet"`
	AppVersion      string              `json:"version"`
	InstanceGuid    string              `json:"instance"`
	InstanceIndex   int                 `json:"index"`
	Reason          DropletExitedReason `json:"reason"`
	ExitStatusCode  int                 `json:"exit_status"`
	ExitDescription string              `json:"exit_description"`
	CrashTimestamp  int64               `json:"crash_timestamp,omitempty"`
}

func NewDropletExitedFromJSON(encoded []byte) (DropletExited, error) {
	dropletExited := DropletExited{}
	err := json.Unmarshal(encoded, &dropletExited)
	if err != nil {
		return DropletExited{}, err
	}
	return dropletExited, nil
}

func (dropletExited DropletExited) ToJSON() []byte {
	result, _ := json.Marshal(dropletExited)
	return result
}

func (dropletExited DropletExited) LogDescription() map[string]string {
	return map[string]string{
		"AppGuid":         dropletExited.AppGuid,
		"AppVersion":      dropletExited.AppVersion,
		"InstanceGuid":    dropletExited.InstanceGuid,
		"InstanceIndex":   strconv.Itoa(dropletExited.InstanceIndex),
		"Reason":          string(dropletExited.Reason),
		"ExitStatusCode":  strconv.Itoa(dropletExited.ExitStatusCode),
		"ExitDescription": dropletExited.ExitDescription,
	}
}
