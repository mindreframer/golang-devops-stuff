package access_log

import (
	"fmt"
	"io"
	"regexp"
	"strconv"

	steno "github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/emitter"
)

type FileAndLoggregatorAccessLogger struct {
	emitter emitter.Emitter
	channel chan AccessLogRecord
	stopCh  chan struct{}
	writer  io.Writer
}

func NewEmitter(loggregatorUrl, loggregatorSharedSecret string, index uint) (emitter.Emitter, error) {
	if !isValidUrl(loggregatorUrl) {
		return nil, fmt.Errorf("Invalid loggregator url %s", loggregatorUrl)
	}
	return emitter.NewEmitter(loggregatorUrl, "RTR", strconv.FormatUint(uint64(index), 10), loggregatorSharedSecret,
		steno.NewLogger("router.loggregator"))
}

func NewFileAndLoggregatorAccessLogger(f io.Writer, e emitter.Emitter) *FileAndLoggregatorAccessLogger {
	a := &FileAndLoggregatorAccessLogger{
		emitter: e,
		writer:  f,
		channel: make(chan AccessLogRecord, 128),
		stopCh:  make(chan struct{}),
	}

	return a
}

func (x *FileAndLoggregatorAccessLogger) Run() {
	for {
		select {
		case record := <-x.channel:
			if x.writer != nil {
				record.WriteTo(x.writer)
			}
			if x.emitter != nil && record.ApplicationId() != "" {
				x.emitter.Emit(record.ApplicationId(), record.LogMessage())
			}
		case <-x.stopCh:
			return
		}
	}
}

func (x *FileAndLoggregatorAccessLogger) Stop() {
	close(x.stopCh)
}

func (x *FileAndLoggregatorAccessLogger) Log(r AccessLogRecord) {
	x.channel <- r
}

var ipAddressRegex, _ = regexp.Compile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(:[0-9]{1,5}){1}$`)
var hostnameRegex, _ = regexp.Compile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])(:[0-9]{1,5}){1}$`)

func isValidUrl(url string) bool {
	return ipAddressRegex.MatchString(url) || hostnameRegex.MatchString(url)
}
