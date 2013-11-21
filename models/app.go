package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type App struct {
	AppGuid    string
	AppVersion string

	Desired            DesiredAppState
	InstanceHeartbeats []InstanceHeartbeat
	CrashCounts        map[int]CrashCount

	instanceHeartbeatsByIndex map[int][]InstanceHeartbeat
}

func NewApp(appGuid string, appVersion string, desired DesiredAppState, instanceHeartbeats []InstanceHeartbeat, crashCounts map[int]CrashCount) *App {
	return &App{
		AppGuid:                   appGuid,
		AppVersion:                appVersion,
		Desired:                   desired,
		InstanceHeartbeats:        instanceHeartbeats,
		CrashCounts:               crashCounts,
		instanceHeartbeatsByIndex: nil,
	}
}

func (a *App) ToJSON() []byte {
	crashCounts := make([]CrashCount, len(a.CrashCounts))
	i := 0
	for _, crashCount := range a.CrashCounts {
		crashCounts[i] = crashCount
		i++
	}

	appForJson := struct {
		AppGuid    string `json:"droplet"`
		AppVersion string `json:"version"`

		Desired            DesiredAppState     `json:"desired"`
		InstanceHeartbeats []InstanceHeartbeat `json:"instance_heartbeats"`
		CrashCounts        []CrashCount        `json:"crash_counts"`
	}{
		a.AppGuid,
		a.AppVersion,
		a.Desired,
		a.InstanceHeartbeats,
		crashCounts,
	}

	result, _ := json.Marshal(appForJson)
	return result
}

func (a *App) LogDescription() map[string]string {
	var desired string
	if a.IsDesired() {
		desired = fmt.Sprintf(`{"NumberOfInstances":%d,"State":"%s","PackageState":"%s"}`, a.Desired.NumberOfInstances, a.Desired.State, a.Desired.PackageState)
	} else {
		desired = "None"
	}

	instanceHeartbeats := []string{}
	for _, heartbeat := range a.InstanceHeartbeats {
		instanceHeartbeats = append(instanceHeartbeats, fmt.Sprintf(`{"InstanceGuid":"%s","InstanceIndex":%d,"State":"%s"}`, heartbeat.InstanceGuid, heartbeat.InstanceIndex, heartbeat.State))
	}

	crashCounts := []string{}
	for _, crashCount := range a.CrashCounts {
		crashCounts = append(crashCounts, fmt.Sprintf(`{"InstanceIndex":%d, "CrashCount":%d}`, crashCount.InstanceIndex, crashCount.CrashCount))
	}

	return map[string]string{
		"AppGuid":            a.AppGuid,
		"AppVersion":         a.AppVersion,
		"Desired":            desired,
		"InstanceHeartbeats": "[" + strings.Join(instanceHeartbeats, ",") + "]",
		"CrashCounts":        "[" + strings.Join(crashCounts, ",") + "]",
	}
}

func (a *App) IsStaged() bool {
	return a.Desired.PackageState == AppPackageStateStaged
}

func (a *App) IsDesired() bool {
	return a.Desired.AppGuid != ""
}

func (a *App) NumberOfDesiredInstances() int {
	return a.Desired.NumberOfInstances
}

func (a *App) IsIndexDesired(index int) bool {
	return index < a.NumberOfDesiredInstances()
}

func (a *App) InstanceWithGuid(instanceGuid string) InstanceHeartbeat {
	for _, heartbeat := range a.InstanceHeartbeats {
		if heartbeat.InstanceGuid == instanceGuid {
			return heartbeat
		}
	}

	return InstanceHeartbeat{}
}

func (a *App) ExtraStartingOrRunningInstances() (extras []InstanceHeartbeat) {
	for _, heartbeat := range a.InstanceHeartbeats {
		if !a.IsIndexDesired(heartbeat.InstanceIndex) && heartbeat.IsStartingOrRunning() {
			extras = append(extras, heartbeat)
		}
	}

	return extras
}

func (a *App) HasStartingOrRunningInstances() bool {
	for _, heartbeat := range a.InstanceHeartbeats {
		if heartbeat.IsStartingOrRunning() {
			return true
		}
	}
	return false
}

func (a *App) NumberOfDesiredIndicesWithAStartingOrRunningInstance() (count int) {
	for index := 0; a.IsIndexDesired(index); index++ {
		if a.HasStartingOrRunningInstanceAtIndex(index) {
			count++
		}
	}

	return count
}

func (a *App) StartingOrRunningInstancesAtIndex(index int) (instances []InstanceHeartbeat) {
	for _, heartbeat := range a.InstanceHeartbeatsAtIndex(index) {
		if heartbeat.IsStartingOrRunning() {
			instances = append(instances, heartbeat)
		}
	}

	return instances
}

func (a *App) HeartbeatsByIndex() (heartbeatsByIndex map[int][]InstanceHeartbeat) {
	heartbeatsByIndex = make(map[int][]InstanceHeartbeat)

	for _, heartbeat := range a.InstanceHeartbeats {
		heartbeatsByIndex[heartbeat.InstanceIndex] = append(heartbeatsByIndex[heartbeat.InstanceIndex], heartbeat)
	}

	return
}

func (a *App) EvacuatingInstancesAtIndex(index int) (instances []InstanceHeartbeat) {
	for _, heartbeat := range a.InstanceHeartbeats {
		if heartbeat.IsEvacuating() && heartbeat.InstanceIndex == index {
			instances = append(instances, heartbeat)
		}
	}

	return
}

func (a *App) HasRunningInstanceAtIndex(index int) bool {
	for _, heartbeat := range a.InstanceHeartbeatsAtIndex(index) {
		if heartbeat.IsRunning() {
			return true
		}
	}

	return false
}

func (a *App) HasStartingInstanceAtIndex(index int) bool {
	for _, heartbeat := range a.InstanceHeartbeatsAtIndex(index) {
		if heartbeat.IsStarting() {
			return true
		}
	}

	return false
}

func (a *App) HasStartingOrRunningInstanceAtIndex(index int) bool {
	for _, heartbeat := range a.InstanceHeartbeatsAtIndex(index) {
		if heartbeat.IsStartingOrRunning() {
			return true
		}
	}

	return false
}

func (a *App) HasCrashedInstanceAtIndex(index int) bool {
	for _, heartbeat := range a.InstanceHeartbeatsAtIndex(index) {
		if heartbeat.IsCrashed() {
			return true
		}
	}

	return false
}

func (a *App) CrashCountAtIndex(instanceIndex int, currentTime time.Time) CrashCount {
	crashCount, found := a.CrashCounts[instanceIndex]
	if !found {
		return CrashCount{
			AppGuid:       a.AppGuid,
			AppVersion:    a.AppVersion,
			InstanceIndex: instanceIndex,
			CreatedAt:     currentTime.Unix(),
		}
	} else {
		return crashCount
	}
}

func (a *App) NumberOfDesiredIndicesReporting() (count int) {
	for index := 0; a.IsIndexDesired(index); index++ {
		if len(a.InstanceHeartbeatsAtIndex(index)) > 0 {
			count++
		}
	}

	return count
}

func (a *App) NumberOfStartingOrRunningInstances() (count int) {
	for _, heartbeat := range a.InstanceHeartbeats {
		if heartbeat.IsStartingOrRunning() {
			count++
		}
	}

	return count
}

func (a *App) NumberOfCrashedInstances() (count int) {
	for _, heartbeat := range a.InstanceHeartbeats {
		if heartbeat.IsCrashed() {
			count++
		}
	}

	return count
}

func (a *App) NumberOfCrashedIndices() (count int) {
	a.verifyInstanceHeartbeatsByIndexIsReady()
	for index := range a.instanceHeartbeatsByIndex {
		if a.HasCrashedInstanceAtIndex(index) && !a.HasStartingOrRunningInstanceAtIndex(index) {
			count++
		}
	}

	return count
}

func (a *App) InstanceHeartbeatsAtIndex(index int) (heartbeats []InstanceHeartbeat) {
	a.verifyInstanceHeartbeatsByIndexIsReady()
	return a.instanceHeartbeatsByIndex[index]
}

func (a *App) verifyInstanceHeartbeatsByIndexIsReady() {
	if a.instanceHeartbeatsByIndex == nil {
		a.instanceHeartbeatsByIndex = make(map[int][]InstanceHeartbeat)
		for _, heartbeat := range a.InstanceHeartbeats {
			a.instanceHeartbeatsByIndex[heartbeat.InstanceIndex] = append(a.instanceHeartbeatsByIndex[heartbeat.InstanceIndex], heartbeat)
		}
	}
}
