package log

import (
	"github.com/cloudfoundry/gorouter/common"
	"github.com/cloudfoundry/gorouter/config"
	steno "github.com/cloudfoundry/gosteno"
	"os"
)

var logger *steno.Logger

var Counter = common.NewLogCounter()

func init() {
	stenoConfig := &steno.Config{
		Sinks: []steno.Sink{steno.NewIOSink(os.Stderr)},
		Codec: steno.NewJsonCodec(),
		Level: steno.LOG_ALL,
	}

	steno.Init(stenoConfig)
	logger = steno.NewLogger("router.init")
}

func SetupLoggerFromConfig(c *config.Config) {
	l, err := steno.GetLogLevel(c.Logging.Level)
	if err != nil {
		panic(err)
	}

	s := make([]steno.Sink, 0)
	if c.Logging.File != "" {
		s = append(s, steno.NewFileSink(c.Logging.File))
	} else {
		s = append(s, steno.NewIOSink(os.Stdout))
	}

	if c.Logging.Syslog != "" {
		s = append(s, steno.NewSyslogSink(c.Logging.Syslog))
	}

	s = append(s, Counter)

	stenoConfig := &steno.Config{
		Sinks: s,
		Codec: steno.NewJsonCodec(),
		Level: l,
	}

	steno.Init(stenoConfig)
	logger = steno.NewLogger("router.global")
}

func Fatal(msg string) { logger.Fatal(msg) }
func Error(msg string) { logger.Error(msg) }
func Warn(msg string)  { logger.Warn(msg) }
func Info(msg string)  { logger.Info(msg) }
func Debug(msg string) { logger.Debug(msg) }

func Fatald(data map[string]interface{}, msg string) { logger.Fatald(data, msg) }
func Errord(data map[string]interface{}, msg string) { logger.Errord(data, msg) }
func Warnd(data map[string]interface{}, msg string)  { logger.Warnd(data, msg) }
func Infod(data map[string]interface{}, msg string)  { logger.Infod(data, msg) }
func Debugd(data map[string]interface{}, msg string) { logger.Debugd(data, msg) }

func Fatalf(msg string, vals ...interface{}) { logger.Fatalf(msg, vals...) }
func Errorf(msg string, vals ...interface{}) { logger.Errorf(msg, vals...) }
func Warnf(msg string, vals ...interface{})  { logger.Warnf(msg, vals...) }
func Infof(msg string, vals ...interface{})  { logger.Infof(msg, vals...) }
func Debugf(msg string, vals ...interface{}) { logger.Debugf(msg, vals...) }
