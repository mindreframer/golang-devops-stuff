package custommatchers

import (
	"github.com/cloudfoundry/hm9000/models"
	"github.com/onsi/gomega"

	"fmt"
)

func EqualPendingStartMessage(expected models.PendingStartMessage) gomega.OmegaMatcher {
	return &pendingStartMessageMatcher{expected: expected}
}

type pendingStartMessageMatcher struct {
	expected models.PendingStartMessage
}

func (m *pendingStartMessageMatcher) Match(actual interface{}) (success bool, message string, err error) {
	actualStartMessage, ok := actual.(models.PendingStartMessage)
	if !ok {
		return false, "", fmt.Errorf("DesiredStateMatcher expects a PendingStartMessage, got %T instead", actual)
	}

	if m.expected.Equal(actualStartMessage) {
		return true, fmt.Sprintf("Expected\n\t%#v\nnot to equal\n\t%#v", actualStartMessage, m.expected), nil
	} else {
		return false, fmt.Sprintf("Expected\n\t%#v\nto equal\n\t%#v", actualStartMessage, m.expected), nil
	}
}

func EqualPendingStopMessage(expected models.PendingStopMessage) gomega.OmegaMatcher {
	return &pendingStopMessageMatcher{expected: expected}
}

type pendingStopMessageMatcher struct {
	expected models.PendingStopMessage
}

func (m *pendingStopMessageMatcher) Match(actual interface{}) (success bool, message string, err error) {
	actualStopMessage, ok := actual.(models.PendingStopMessage)
	if !ok {
		return false, "", fmt.Errorf("DesiredStateMatcher expects a PendingStopMessage, got %T instead", actual)
	}

	if m.expected.Equal(actualStopMessage) {
		return true, fmt.Sprintf("Expected\n\t%#v\nnot to equal\n\t%#v", actualStopMessage, m.expected), nil
	} else {
		return false, fmt.Sprintf("Expected\n\t%#v\nto equal\n\t%#v", actualStopMessage, m.expected), nil
	}
}
