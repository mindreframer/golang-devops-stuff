package appfixture

import (
	. "github.com/cloudfoundry/hm9000/models"
)

type AppFixture struct {
	AppGuid    string
	AppVersion string

	instances map[int]Instance
	DeaGuid   string
}

type Instance struct {
	InstanceGuid  string
	InstanceIndex int
	AppGuid       string
	AppVersion    string
	DeaGuid       string
}

func NewAppFixture() AppFixture {
	return newAppForDeaGuid(Guid())
}

func newAppForDeaGuid(deaGuid string) AppFixture {
	return AppFixture{
		AppGuid:    Guid(),
		AppVersion: Guid(),
		instances:  make(map[int]Instance, 0),
		DeaGuid:    deaGuid,
	}
}

func (app AppFixture) CrashedInstanceHeartbeatAtIndex(index int) InstanceHeartbeat {
	return InstanceHeartbeat{
		State:         InstanceStateCrashed,
		AppGuid:       app.AppGuid,
		AppVersion:    app.AppVersion,
		InstanceGuid:  Guid(),
		InstanceIndex: index,
		DeaGuid:       app.DeaGuid,
	}
}

func (app AppFixture) InstanceAtIndex(index int) Instance {
	_, ok := app.instances[index]
	if !ok {
		app.instances[index] = Instance{
			InstanceGuid:  Guid(),
			InstanceIndex: index,
			AppGuid:       app.AppGuid,
			AppVersion:    app.AppVersion,
			DeaGuid:       app.DeaGuid,
		}
	}

	return app.instances[index]
}

func (app AppFixture) DesiredState(numberOfInstances int) DesiredAppState {
	return DesiredAppState{
		AppGuid:           app.AppGuid,
		AppVersion:        app.AppVersion,
		NumberOfInstances: numberOfInstances,
		State:             AppStateStarted,
		PackageState:      AppPackageStateStaged,
	}
}

func (instance Instance) Heartbeat() InstanceHeartbeat {
	return InstanceHeartbeat{
		AppGuid:       instance.AppGuid,
		AppVersion:    instance.AppVersion,
		InstanceGuid:  instance.InstanceGuid,
		InstanceIndex: instance.InstanceIndex,
		State:         InstanceStateRunning,
		DeaGuid:       instance.DeaGuid,
	}
}

func (instance Instance) DropletExited(reason DropletExitedReason) DropletExited {
	droplet_exited := DropletExited{
		CCPartition:     "default",
		AppGuid:         instance.AppGuid,
		AppVersion:      instance.AppVersion,
		InstanceGuid:    instance.InstanceGuid,
		InstanceIndex:   instance.InstanceIndex,
		Reason:          reason,
		ExitDescription: "exited",
	}

	return droplet_exited
}

func (app AppFixture) Heartbeat(instances int) Heartbeat {
	instanceHeartbeats := make([]InstanceHeartbeat, instances)
	for i := 0; i < instances; i++ {
		instanceHeartbeats[i] = app.InstanceAtIndex(i).Heartbeat()
	}

	return Heartbeat{
		DeaGuid:            app.DeaGuid,
		InstanceHeartbeats: instanceHeartbeats,
	}
}

func (app AppFixture) DropletUpdated() DropletUpdatedMessage {
	return DropletUpdatedMessage{
		AppGuid: app.AppGuid,
	}
}
