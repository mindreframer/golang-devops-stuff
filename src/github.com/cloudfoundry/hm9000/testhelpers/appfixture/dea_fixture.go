package appfixture

import (
	"github.com/cloudfoundry/hm9000/models"
)

type DeaFixture struct {
	DeaGuid string
	apps    map[int]AppFixture
}

func NewDeaFixture() DeaFixture {
	return DeaFixture{
		DeaGuid: models.Guid(),
		apps:    make(map[int]AppFixture, 0),
	}
}

func (dea DeaFixture) GetApp(index int) AppFixture {
	_, ok := dea.apps[index]
	if !ok {
		dea.apps[index] = newAppForDeaGuid(dea.DeaGuid)
	}

	return dea.apps[index]
}

func (dea DeaFixture) Heartbeat(numApps int) models.Heartbeat {
	instanceHeartbeats := make([]models.InstanceHeartbeat, 0)
	for i := 0; i < numApps; i++ {
		instanceHeartbeats = append(instanceHeartbeats, dea.GetApp(i).InstanceAtIndex(0).Heartbeat())
	}

	return models.Heartbeat{
		DeaGuid:            dea.DeaGuid,
		InstanceHeartbeats: instanceHeartbeats,
	}
}
