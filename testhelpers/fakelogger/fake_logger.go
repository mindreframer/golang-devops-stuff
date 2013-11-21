package fakelogger

import (
	"encoding/json"
	"fmt"
)

type FakeLogger struct {
	LoggedSubjects []string
	LoggedErrors   []error
	LoggedMessages []string
}

func NewFakeLogger() *FakeLogger {
	return &FakeLogger{
		LoggedSubjects: []string{},
		LoggedErrors:   []error{},
		LoggedMessages: []string{},
	}
}

func (logger *FakeLogger) Info(subject string, messages ...map[string]string) {
	logger.LoggedSubjects = append(logger.LoggedSubjects, subject)
	logger.LoggedMessages = append(logger.LoggedMessages, logger.squashedMessage(messages...))
}

func (logger *FakeLogger) Debug(subject string, messages ...map[string]string) {
	logger.LoggedSubjects = append(logger.LoggedSubjects, subject)
	logger.LoggedMessages = append(logger.LoggedMessages, logger.squashedMessage(messages...))
}

func (logger *FakeLogger) Error(subject string, err error, messages ...map[string]string) {
	logger.LoggedSubjects = append(logger.LoggedSubjects, subject)
	logger.LoggedErrors = append(logger.LoggedErrors, err)
	logger.LoggedMessages = append(logger.LoggedMessages, logger.squashedMessage(messages...))
}

func (logger *FakeLogger) squashedMessage(messages ...map[string]string) (squashed string) {
	for _, message := range messages {
		encoded, err := json.Marshal(message)
		if err != nil {
			panic(fmt.Sprintf("LOGGER GOT AN UNMARSHALABLE MESSAGE: %s", err.Error()))
		}
		squashed += " - " + string(encoded)
	}
	return
}
