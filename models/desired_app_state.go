package models

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

//Desired app state
type AppState string

const (
	AppStateInvalid AppState = ""
	AppStateStarted AppState = "STARTED"
	AppStateStopped AppState = "STOPPED"
)

type AppPackageState string

const (
	AppPackageStateInvalid AppPackageState = ""
	AppPackageStateFailed  AppPackageState = "FAILED"
	AppPackageStatePending AppPackageState = "PENDING"
	AppPackageStateStaged  AppPackageState = "STAGED"
)

type DesiredAppState struct {
	AppGuid           string          `json:"id"`
	AppVersion        string          `json:"version"`
	NumberOfInstances int             `json:"instances"`
	State             AppState        `json:"state"`
	PackageState      AppPackageState `json:"package_state"`
}

func NewDesiredAppStateFromJSON(encoded []byte) (DesiredAppState, error) {
	var desired DesiredAppState
	err := json.Unmarshal(encoded, &desired)
	if err != nil {
		return DesiredAppState{}, err
	}
	return desired, nil
}

func NewDesiredAppStateFromCSV(appGuid, appVersion string, encoded []byte) (DesiredAppState, error) {
	values := strings.Split(string(encoded), ",")

	if len(values) != 3 {
		return DesiredAppState{}, fmt.Errorf("invalid desired state (need 3 values, have %d)", len(values))
	}

	numberOfInstances, err := strconv.Atoi(values[0])
	if err != nil {
		return DesiredAppState{}, err
	}

	return DesiredAppState{
		AppGuid:           appGuid,
		AppVersion:        appVersion,
		NumberOfInstances: numberOfInstances,
		State:             AppState(values[1]),
		PackageState:      AppPackageState(values[2]),
	}, nil
}

func (state DesiredAppState) ToJSON() []byte {
	result, _ := json.Marshal(state)
	return result
}

func (state DesiredAppState) ToCSV() []byte {
	return []byte(fmt.Sprintf("%d,%s,%s", state.NumberOfInstances, state.State, state.PackageState))
}

func (state DesiredAppState) LogDescription() map[string]string {
	return map[string]string{
		"AppGuid":           state.AppGuid,
		"AppVersion":        state.AppVersion,
		"NumberOfInstances": strconv.Itoa(state.NumberOfInstances),
		"State":             string(state.State),
		"PackageState":      string(state.PackageState),
	}
}

func (state DesiredAppState) Equal(other DesiredAppState) bool {
	return state.AppGuid == other.AppGuid &&
		state.AppVersion == other.AppVersion &&
		state.NumberOfInstances == other.NumberOfInstances &&
		state.State == other.State &&
		state.PackageState == other.PackageState
}

func (state DesiredAppState) StoreKey() string {
	return state.AppGuid + "," + state.AppVersion
}
