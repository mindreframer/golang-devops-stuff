package router

import (
	"os/exec"
	"time"

	"github.com/cloudfoundry/yagnats"
	. "launchpad.net/gocheck"

	"github.com/cloudfoundry/gorouter/config"
	"github.com/cloudfoundry/gorouter/log"
	"github.com/cloudfoundry/gorouter/route"
	"github.com/cloudfoundry/gorouter/test"
)

type IntegrationSuite struct {
	Config     *config.Config
	mbusClient yagnats.NATSClient
	router     *Router

	natsPort uint16
	natsCmd  *exec.Cmd
}

var _ = Suite(&IntegrationSuite{})

func (s *IntegrationSuite) SetUpTest(c *C) {
	port := nextAvailPort()

	s.startNats(port)
}

func (s *IntegrationSuite) TearDownTest(c *C) {
	if s.natsCmd != nil {
		s.stopNats()
	}
}

func (s *IntegrationSuite) TestNatsConnectivity(c *C) {
	proxyPort := nextAvailPort()
	statusPort := nextAvailPort()

	s.Config = SpecConfig(s.natsPort, statusPort, proxyPort)

	// ensure the threshold is longer than the interval that we check,
	// because we set the route's timestamp to time.Now() on the interval
	// as part of pausing
	s.Config.PruneStaleDropletsInterval = 1 * time.Second
	s.Config.DropletStaleThreshold = 2 * s.Config.PruneStaleDropletsInterval

	log.SetupLoggerFromConfig(s.Config)

	s.router = NewRouter(s.Config)

	s.router.Run()

	s.mbusClient = s.router.mbusClient

	staleCheckInterval := s.Config.PruneStaleDropletsInterval
	staleThreshold := s.Config.DropletStaleThreshold

	s.Config.DropletStaleThreshold = staleThreshold

	zombieApp := test.NewGreetApp([]route.Uri{"zombie.vcap.me"}, proxyPort, s.mbusClient, nil)
	zombieApp.Listen()

	runningApp := test.NewGreetApp([]route.Uri{"innocent.bystander.vcap.me"}, proxyPort, s.mbusClient, nil)
	runningApp.Listen()

	c.Assert(s.waitAppRegistered(zombieApp, 2*time.Second), Equals, true)
	c.Assert(s.waitAppRegistered(runningApp, 2*time.Second), Equals, true)

	heartbeatInterval := 200 * time.Millisecond
	zombieTicker := time.NewTicker(heartbeatInterval)
	runningTicker := time.NewTicker(heartbeatInterval)

	go func() {
		for {
			select {
			case <-zombieTicker.C:
				zombieApp.Register()
			case <-runningTicker.C:
				runningApp.Register()
			}
		}
	}()

	zombieApp.VerifyAppStatus(200, c)

	// Give enough time to register multiple times
	time.Sleep(heartbeatInterval * 3)

	// kill registration ticker => kill app (must be before stopping NATS since app.Register is fake and queues messages in memory)
	zombieTicker.Stop()

	natsPort := s.natsPort
	s.stopNats()

	// Give router time to make a bad decision (i.e. prune routes)
	time.Sleep(staleCheckInterval + staleThreshold + 250*time.Millisecond)

	// While NATS is down no routes should go down
	zombieApp.VerifyAppStatus(200, c)
	runningApp.VerifyAppStatus(200, c)

	s.startNats(natsPort)

	// Right after NATS starts up all routes should stay up
	zombieApp.VerifyAppStatus(200, c)
	runningApp.VerifyAppStatus(200, c)

	zombieGone := make(chan bool)

	go func() {
		for {
			// Finally the zombie is cleaned up. Maybe proactively enqueue Unregister events in DEA's.
			err := zombieApp.CheckAppStatus(404)
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			err = runningApp.CheckAppStatus(200)
			if err != nil {
				time.Sleep(100 * time.Millisecond)
				continue
			}

			zombieGone <- true

			break
		}
	}()

	select {
	case <-zombieGone:
	case <-time.After(staleCheckInterval + staleThreshold + 5*time.Second):
		c.Error("Zombie app was not pruned.")
	}
}

func (s *IntegrationSuite) startNats(port uint16) {
	s.natsPort = port
	s.natsCmd = StartNats(int(port))

	err := waitUntilNatsUp(port)
	if err != nil {
		panic("cannot connect to NATS")
	}
}

func (s *IntegrationSuite) stopNats() {
	StopNats(s.natsCmd)

	err := waitUntilNatsDown(s.natsPort)
	if err != nil {
		panic("cannot shut down NATS")
	}

	s.natsPort = 0
	s.natsCmd = nil
}

func (s *IntegrationSuite) restartNats() {
	port := s.natsPort
	s.stopNats()
	s.startNats(port)
}

func (s *IntegrationSuite) waitMsgReceived(a *test.TestApp, shouldBeRegistered bool, t time.Duration) bool {
	registeredOrUnregistered := make(chan bool)

	go func() {
		for {
			received := true
			for _, v := range a.Urls() {
				_, ok := s.router.registry.Lookup(v)
				if ok != shouldBeRegistered {
					received = false
					break
				}
			}

			if received {
				registeredOrUnregistered <- true
				break
			}

			time.Sleep(50 * time.Millisecond)
		}
	}()

	select {
	case <-registeredOrUnregistered:
		return true
	case <-time.After(t):
		return false
	}
}

func (s *IntegrationSuite) waitAppRegistered(app *test.TestApp, timeout time.Duration) bool {
	return s.waitMsgReceived(app, true, timeout)
}
