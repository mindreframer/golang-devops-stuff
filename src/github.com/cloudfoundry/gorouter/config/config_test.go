package config

import (
	. "launchpad.net/gocheck"
	"time"
)

type ConfigSuite struct {
	*Config
}

var _ = Suite(&ConfigSuite{})

func (s *ConfigSuite) SetUpTest(c *C) {
	s.Config = DefaultConfig()
}

func (s *ConfigSuite) TestStatus(c *C) {
	var b = []byte(`
status:
  port: 1234
  user: user
  pass: pass
`)

	c.Check(s.Status.Port, Equals, uint16(8082))
	c.Check(s.Status.User, Equals, "")
	c.Check(s.Status.Pass, Equals, "")

	s.Config.Initialize(b)

	c.Check(s.Status.Port, Equals, uint16(1234))
	c.Check(s.Status.User, Equals, "user")
	c.Check(s.Status.Pass, Equals, "pass")
}

func (s *ConfigSuite) TestEndpointTimeout(c *C) {
	var b = []byte(`
endpoint_timeout: 10
`)

	c.Check(s.EndpointTimeoutInSeconds, Equals, 60)

	s.Config.Initialize(b)

	c.Check(s.EndpointTimeoutInSeconds, Equals, 10)
}

func (s *ConfigSuite) TestNats(c *C) {
	var b = []byte(`
nats:
  - host: remotehost
    port: 4223
    user: user
    pass: pass
`)

	c.Assert(len(s.Nats), Not(Equals), 0)
	c.Check(s.Nats[0].Host, Equals, "localhost")
	c.Check(s.Nats[0].Port, Equals, uint16(4222))
	c.Check(s.Nats[0].User, Equals, "")
	c.Check(s.Nats[0].Pass, Equals, "")

	s.Config.Initialize(b)

	c.Assert(len(s.Nats), Not(Equals), 0)
	c.Check(s.Nats[0].Host, Equals, "remotehost")
	c.Check(s.Nats[0].Port, Equals, uint16(4223))
	c.Check(s.Nats[0].User, Equals, "user")
	c.Check(s.Nats[0].Pass, Equals, "pass")
}

func (s *ConfigSuite) TestLogging(c *C) {
	var b = []byte(`
logging:
  file: /tmp/file
  syslog: syslog
  level: debug2
`)

	c.Check(s.Logging.File, Equals, "")
	c.Check(s.Logging.Syslog, Equals, "")
	c.Check(s.Logging.Level, Equals, "debug")

	s.Config.Initialize(b)

	c.Check(s.Logging.File, Equals, "/tmp/file")
	c.Check(s.Logging.Syslog, Equals, "syslog")
	c.Check(s.Logging.Level, Equals, "debug2")
}

func (s *ConfigSuite) TestLoggregator(c *C) {
	var b = []byte(`
loggregatorConfig:
  url: 10.10.16.14:3456
`)

	c.Check(s.LoggregatorConfig.Url, Equals, "")

	s.Config.Initialize(b)

	c.Check(s.LoggregatorConfig.Url, Equals, "10.10.16.14:3456")
}

func (s *ConfigSuite) TestConfig(c *C) {
	var b = []byte(`
port: 8082
index: 1
pidfile: /tmp/pidfile
go_max_procs: 2
trace_key: "foo"
access_log: "/tmp/access_log"

publish_start_message_interval: 1
prune_stale_droplets_interval: 2
droplet_stale_threshold: 3
publish_active_apps_interval: 4
start_response_delay_interval: 15
`)

	c.Check(s.Port, Equals, uint16(8081))
	c.Check(s.Index, Equals, uint(0))
	c.Check(s.Pidfile, Equals, "")
	c.Check(s.GoMaxProcs, Equals, 8)
	c.Check(s.TraceKey, Equals, "")
	c.Check(s.AccessLog, Equals, "")

	c.Check(s.PublishStartMessageIntervalInSeconds, Equals, 30)
	c.Check(s.PruneStaleDropletsInterval, Equals, 30*time.Second)
	c.Check(s.DropletStaleThreshold, Equals, 120*time.Second)
	c.Check(s.PublishActiveAppsInterval, Equals, 0*time.Second)
	c.Check(s.StartResponseDelayInterval, Equals, 5*time.Second)

	s.Config.Initialize(b)

	s.Config.Process()

	c.Check(s.Port, Equals, uint16(8082))
	c.Check(s.Index, Equals, uint(1))
	c.Check(s.Pidfile, Equals, "/tmp/pidfile")
	c.Check(s.GoMaxProcs, Equals, 2)
	c.Check(s.TraceKey, Equals, "foo")
	c.Check(s.AccessLog, Equals, "/tmp/access_log")

	c.Check(s.PublishStartMessageIntervalInSeconds, Equals, 1)
	c.Check(s.PruneStaleDropletsInterval, Equals, 2*time.Second)
	c.Check(s.DropletStaleThreshold, Equals, 3*time.Second)
	c.Check(s.PublishActiveAppsInterval, Equals, 4*time.Second)
	c.Check(s.StartResponseDelayInterval, Equals, 15*time.Second)
}
