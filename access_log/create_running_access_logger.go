package access_log

import (
	"github.com/cloudfoundry/gorouter/config"
	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/emitter"

	"os"
)

func CreateRunningAccessLogger(config *config.Config) (AccessLogger, error) {
	loggregatorUrl := config.LoggregatorConfig.Url

	if config.AccessLog == "" && loggregatorUrl == "" {
		return &NullAccessLogger{}, nil
	}

	logger := steno.NewLogger("access_log")

	var err error
	var file *os.File
	if config.AccessLog != "" {
		file, err = os.OpenFile(config.AccessLog, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			logger.Errorf("Error creating accesslog file, %s: (%s)", config.AccessLog, err.Error())
			return nil, err
		}
	}

	var e emitter.Emitter
	if loggregatorUrl != "" {
		loggregatorSharedSecret := config.LoggregatorConfig.SharedSecret
		e, err = NewEmitter(loggregatorUrl, loggregatorSharedSecret, config.Index)
		if err != nil {
			logger.Errorf("Error creating loggregator emitter: (%s)", err.Error())
			return nil, err
		}
	}

	accessLogger := NewFileAndLoggregatorAccessLogger(file, e)
	go accessLogger.Run()
	return accessLogger, nil
}
