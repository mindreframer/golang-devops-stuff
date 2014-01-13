package access_log

import (
	"io"
	"regexp"
	"strconv"

	"github.com/cloudfoundry/gorouter/log"

	"github.com/cloudfoundry/loggregatorlib/emitter"
	steno "github.com/cloudfoundry/gosteno"
)

type FileAndLoggregatorAccessLogger struct {
	emitter  emitter.Emitter
	channel  chan AccessLogRecord
	writer   io.Writer
	index    uint
}

func NewFileAndLoggregatorAccessLogger(f io.Writer, loggregatorUrl, loggregatorSharedSecret string, index uint) *FileAndLoggregatorAccessLogger {
	a := &FileAndLoggregatorAccessLogger{
		writer:     f,
		channel:     make(chan AccessLogRecord, 128),
		index: index,
	}

	if isValidUrl(loggregatorUrl) {
		a.emitter, _ = emitter.NewEmitter(loggregatorUrl, "RTR", strconv.FormatUint(uint64(index), 10), loggregatorSharedSecret, steno.NewLogger("router.loggregator"))
	} else {
		log.Errorf("Invalid loggregator url %s", loggregatorUrl)
	}

	return a
}

func (x *FileAndLoggregatorAccessLogger) Run() {
	for access_record := range x.channel {
		if x.writer != nil {
			access_record.WriteTo(x.writer)
		}
		if x.emitter != nil {
			access_record.Emit(x.emitter)
		}
	}
}

func (x *FileAndLoggregatorAccessLogger) Stop() {
	close(x.channel)
}

func (x *FileAndLoggregatorAccessLogger) Log(r AccessLogRecord) {
	x.channel <- r
}

var ipAddressRegex, _ = regexp.Compile(`^(([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])\.){3}([0-9]|[1-9][0-9]|1[0-9]{2}|2[0-4][0-9]|25[0-5])(:[0-9]{1,5}){1}$`)
var hostnameRegex, _ = regexp.Compile(`^(([a-zA-Z0-9]|[a-zA-Z0-9][a-zA-Z0-9\-]*[a-zA-Z0-9])\.)*([A-Za-z0-9]|[A-Za-z0-9][A-Za-z0-9\-]*[A-Za-z0-9])(:[0-9]{1,5}){1}$`)

func isValidUrl(url string) bool {
	return ipAddressRegex.MatchString(url) || hostnameRegex.MatchString(url)
}
