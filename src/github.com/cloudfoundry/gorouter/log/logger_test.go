package log

import (
	steno "github.com/cloudfoundry/gosteno"
	. "launchpad.net/gocheck"

	"github.com/cloudfoundry/gorouter/config"
)

type LoggerSuite struct{}

var _ = Suite(&LoggerSuite{})

func (s *LoggerSuite) TestSetupLoggerFromConfig(c *C) {
	cfg := config.DefaultConfig()
	cfg.Logging.File = "/tmp/gorouter.log"

	SetupLoggerFromConfig(cfg)

	count := Counter.GetCount("info")
	logger := steno.NewLogger("test")
	logger.Info("Hello")
	c.Assert(Counter.GetCount("info"), Equals, count+1)
}
