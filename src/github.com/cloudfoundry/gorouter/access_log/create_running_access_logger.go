package access_log

import (
	"os"

	"github.com/cloudfoundry/gorouter/config"
)

func CreateRunningAccessLogger(config *config.Config) (accessLogger AccessLogger) {
	loggregatorUrl := config.LoggregatorConfig.Url
	loggregatorSharedSecret := config.LoggregatorConfig.SharedSecret

	if config.AccessLog != "" || loggregatorUrl != "" {
		file, err := os.OpenFile(config.AccessLog, os.O_WRONLY | os.O_APPEND | os.O_CREATE, 0666)
		if err != nil && config.AccessLog != "" {
			panic(err)
		}

		accessLogger = NewFileAndLoggregatorAccessLogger(file, loggregatorUrl, loggregatorSharedSecret, config.Index)
		go accessLogger.Run()
	} else {
		accessLogger = &NullAccessLogger{}
	}

	return
}
