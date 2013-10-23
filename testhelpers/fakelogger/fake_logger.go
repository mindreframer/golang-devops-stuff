package fakelogger

import (
	"encoding/json"
	"fmt"
)

type FakeLogger struct {
	LoggedSubjects []string
	LoggedErrors   []error
	LoggedMessages [][]map[string]string
}

func NewFakeLogger() *FakeLogger {
	return &FakeLogger{
		LoggedSubjects: []string{},
		LoggedErrors:   []error{},
		LoggedMessages: [][]map[string]string{},
	}
}

func (logger *FakeLogger) Info(subject string, messages ...map[string]string) {
	for _, message := range messages {
		_, err := json.Marshal(message)
		if err != nil {
			panic(fmt.Sprintf("LOGGER GOT AN UNMARSHALABLE MESSAGE: %s", err.Error()))
		}
	}
	logger.LoggedSubjects = append(logger.LoggedSubjects, subject)
	logger.LoggedMessages = append(logger.LoggedMessages, messages)
}

func (logger *FakeLogger) Error(subject string, err error, messages ...map[string]string) {
	for _, message := range messages {
		_, err := json.Marshal(message)
		if err != nil {
			panic(fmt.Sprintf("LOGGER GOT AN UNMARSHALABLE MESSAGE: %s", err.Error()))
		}
	}
	logger.LoggedSubjects = append(logger.LoggedSubjects, subject)
	logger.LoggedErrors = append(logger.LoggedErrors, err)
	logger.LoggedMessages = append(logger.LoggedMessages, messages)
}
