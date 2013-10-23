package logger

import (
	"encoding/json"
	"github.com/cloudfoundry/gosteno"
	"log"
	"os"
)

type Logger interface {
	Info(subject string, messages ...map[string]string)
	Error(subject string, err error, messages ...map[string]string)
}

type RealLogger struct {
	steno  *gosteno.Logger
	logger *log.Logger
}

func NewRealLogger(steno *gosteno.Logger) *RealLogger {
	return &RealLogger{
		steno:  steno,
		logger: log.New(os.Stdout, "", log.LstdFlags),
	}
}

func (logger *RealLogger) Info(subject string, messages ...map[string]string) {
	logger.print(subject, logger.parseMessages(messages))
}

func (logger *RealLogger) Error(subject string, err error, messages ...map[string]string) {
	logger.print(subject, " - Error: "+err.Error()+logger.parseMessages(messages))
}

func (logger *RealLogger) parseMessages(messages []map[string]string) string {
	messageString := ""
	for _, message := range messages {
		messageBytes, _ := json.Marshal(message)
		messageString += " - " + string(messageBytes)
	}

	return messageString
}

func (logger *RealLogger) print(subject string, message string) {
	logger.logger.Printf("%s%s", subject, message)
	logger.steno.Info(subject + message)
}
