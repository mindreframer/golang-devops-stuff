package logger

import (
	"encoding/json"
	"github.com/cloudfoundry/gosteno"
)

type Logger interface {
	Info(subject string, messages ...map[string]string)
	Debug(subject string, messages ...map[string]string)
	Error(subject string, err error, messages ...map[string]string)
}

type RealLogger struct {
	steno *gosteno.Logger
}

func NewRealLogger(steno *gosteno.Logger) *RealLogger {
	return &RealLogger{
		steno: steno,
	}
}

func (logger *RealLogger) Debug(subject string, messages ...map[string]string) {
	logger.steno.Debug(subject + logger.parseMessages(messages))
}

func (logger *RealLogger) Info(subject string, messages ...map[string]string) {
	logger.steno.Info(subject + logger.parseMessages(messages))
}

func (logger *RealLogger) Error(subject string, err error, messages ...map[string]string) {
	logger.steno.Error(subject + " - Error:" + err.Error() + logger.parseMessages(messages))
}

func (logger *RealLogger) parseMessages(messages []map[string]string) string {
	messageString := ""
	for _, message := range messages {
		messageBytes, _ := json.Marshal(message)
		messageString += " - " + string(messageBytes)
	}

	return messageString
}
