package models

import (
	"encoding/json"
	"strconv"
)

type CrashCount struct {
	AppGuid       string `json:"droplet"`
	AppVersion    string `json:"version"`
	InstanceIndex int    `json:"instance_index"`
	CrashCount    int    `json:"crash_count"`
	CreatedAt     int64  `json:"created_at"`
}

func NewCrashCountFromJSON(encoded []byte) (CrashCount, error) {
	crashCount := CrashCount{}
	err := json.Unmarshal(encoded, &crashCount)
	if err != nil {
		return CrashCount{}, err
	}
	return crashCount, nil
}

func (crashCount CrashCount) ToJSON() []byte {
	result, _ := json.Marshal(crashCount)
	return result
	return nil
}

func (crashCount CrashCount) StoreKey() string {
	return crashCount.AppGuid + "-" + crashCount.AppVersion + "-" + strconv.Itoa(crashCount.InstanceIndex)
}
