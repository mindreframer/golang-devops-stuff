package mcat_test

import (
	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/hm9000/testhelpers/desiredstateserver"
	"github.com/cloudfoundry/storeadapter/storerunner"
	"github.com/cloudfoundry/yagnats"
	. "github.com/onsi/gomega"
)

type Simulator struct {
	conf                   *config.Config
	storeRunner            storerunner.StoreRunner
	store                  store.Store
	desiredStateServer     *desiredstateserver.DesiredStateServer
	currentHeartbeats      []models.Heartbeat
	currentTimestamp       int
	cliRunner              *CLIRunner
	TicksToAttainFreshness int
	TicksToExpireHeartbeat int
	GracePeriod            int
	messageBus             yagnats.NATSClient
}

func NewSimulator(conf *config.Config, storeRunner storerunner.StoreRunner, store store.Store, desiredStateServer *desiredstateserver.DesiredStateServer, cliRunner *CLIRunner, messageBus yagnats.NATSClient) *Simulator {
	desiredStateServer.Reset()

	return &Simulator{
		currentTimestamp:       100,
		conf:                   conf,
		storeRunner:            storeRunner,
		store:                  store,
		desiredStateServer:     desiredStateServer,
		cliRunner:              cliRunner,
		TicksToAttainFreshness: int(conf.ActualFreshnessTTLInHeartbeats) + 1,
		TicksToExpireHeartbeat: int(conf.HeartbeatTTLInHeartbeats),
		GracePeriod:            int(conf.GracePeriodInHeartbeats),
		messageBus:             messageBus,
	}
}

func (s *Simulator) Tick(numTicks int) {
	timeBetweenTicks := int(s.conf.HeartbeatPeriod)

	for i := 0; i < numTicks; i++ {
		s.currentTimestamp += timeBetweenTicks
		s.storeRunner.FastForwardTime(timeBetweenTicks)
		s.sendHeartbeats()
		s.cliRunner.Run("fetch_desired", s.currentTimestamp)
		s.cliRunner.Run("analyze", s.currentTimestamp)
		s.cliRunner.Run("send", s.currentTimestamp)
	}
}

func (s *Simulator) sendHeartbeats() {
	s.store.SaveMetric("SavedHeartbeats", 0)
	s.cliRunner.StartListener(s.currentTimestamp)
	for _, heartbeat := range s.currentHeartbeats {
		s.messageBus.Publish("dea.heartbeat", heartbeat.ToJSON())
	}

	Eventually(func() float64 {
		nHeartbeats, _ := s.store.GetMetric("SavedHeartbeats")
		return nHeartbeats
	}, 5.0, 0.05).Should(BeNumerically("==", len(s.currentHeartbeats)))

	s.cliRunner.StopListener()
}

func (s *Simulator) SetDesiredState(desiredStates ...models.DesiredAppState) {
	s.desiredStateServer.SetDesiredState(desiredStates)
}

func (s *Simulator) SetCurrentHeartbeats(heartbeats ...models.Heartbeat) {
	s.currentHeartbeats = heartbeats
}
