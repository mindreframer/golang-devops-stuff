package emitter

import (
	"code.google.com/p/gogoprotobuf/proto"
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/loggregatorclient"
	"github.com/cloudfoundry/loggregatorlib/logmessage"
	//	"regexp"
	"strings"
	"time"
)

var (
	MAX_MESSAGE_BYTE_SIZE = (9 * 1024) - 512
	TRUNCATED_BYTES       = []byte("TRUNCATED")
	TRUNCATED_OFFSET      = MAX_MESSAGE_BYTE_SIZE - len(TRUNCATED_BYTES)
)

type Emitter interface {
	Emit(string, string)
	EmitError(string, string)
	EmitLogMessage(*logmessage.LogMessage)
}

type LoggregatorEmitter struct {
	LoggregatorClient loggregatorclient.LoggregatorClient
	sn                string
	sId               string
	sharedSecret      string
	logger            *gosteno.Logger
}

func isEmpty(s string) bool {
	return len(strings.TrimSpace(s)) == 0
}

func splitMessage(message string) []string {
	return strings.FieldsFunc(message, func(r rune) bool {
		return r == '\n' || r == '\r'
	})
}

func (e *LoggregatorEmitter) Emit(appid, message string) {
	e.emit(appid, message, logmessage.LogMessage_OUT)
}

func (e *LoggregatorEmitter) EmitError(appid, message string) {
	e.emit(appid, message, logmessage.LogMessage_ERR)
}

func (e *LoggregatorEmitter) emit(appid, message string, messageType logmessage.LogMessage_MessageType) {
	if isEmpty(appid) || isEmpty(message) {
		return
	}
	logMessage := e.newLogMessage(appid, message, messageType)
	e.logger.Debugf("Logging message from %s of type %s with appid %s and with data %s", *logMessage.SourceName, logMessage.MessageType, *logMessage.AppId, string(logMessage.Message))

	e.EmitLogMessage(logMessage)
}

func (e *LoggregatorEmitter) EmitLogMessage(logMessage *logmessage.LogMessage) {
	messages := splitMessage(string(logMessage.GetMessage()))

	for _, message := range messages {
		if isEmpty(message) {
			continue
		}

		if len(message) > MAX_MESSAGE_BYTE_SIZE {
			logMessage.Message = append([]byte(message)[0:TRUNCATED_OFFSET], TRUNCATED_BYTES...)
		} else {
			logMessage.Message = []byte(message)
		}
		if e.sharedSecret == "" {
			marshalledLogMessage, err := proto.Marshal(logMessage)
			if err != nil {
				e.logger.Errorf("Error marshalling message: %s", err)
				return
			}
			e.LoggregatorClient.Send(marshalledLogMessage)
		} else {
			logEnvelope, err := e.newLogEnvelope(*logMessage.AppId, logMessage)
			if err != nil {
				e.logger.Errorf("Error creating envelope: %s", err)
				return
			}
			marshalledLogEnvelope, err := proto.Marshal(logEnvelope)
			if err != nil {
				e.logger.Errorf("Error marshalling envelope: %s", err)
				return
			}
			e.LoggregatorClient.Send(marshalledLogEnvelope)
		}
	}
}

func NewEmitter(loggregatorServer, sourceName, sourceId, sharedSecret string, logger *gosteno.Logger) (*LoggregatorEmitter, error) {
	if logger == nil {
		logger = gosteno.NewLogger("loggregatorlib.emitter")
	}

	e := &LoggregatorEmitter{sharedSecret: sharedSecret}

	e.sn = sourceName
	e.logger = logger
	e.LoggregatorClient = loggregatorclient.NewLoggregatorClient(loggregatorServer, logger, loggregatorclient.DefaultBufferSize)
	e.sId = sourceId

	e.logger.Debugf("Created new loggregator emitter: %#v", e)
	return e, nil
}

func (e *LoggregatorEmitter) newLogMessage(appId, message string, mt logmessage.LogMessage_MessageType) *logmessage.LogMessage {
	currentTime := time.Now()

	return &logmessage.LogMessage{
		Message:     []byte(message),
		AppId:       proto.String(appId),
		MessageType: &mt,
		SourceId:    &e.sId,
		Timestamp:   proto.Int64(currentTime.UnixNano()),
		SourceName:  &e.sn,
	}
}

func (e *LoggregatorEmitter) newLogEnvelope(appId string, message *logmessage.LogMessage) (*logmessage.LogEnvelope, error) {
	envelope := &logmessage.LogEnvelope{
		LogMessage: message,
		RoutingKey: proto.String(appId),
		Signature:  []byte{},
	}
	err := envelope.SignEnvelope(e.sharedSecret)

	return envelope, err
}
