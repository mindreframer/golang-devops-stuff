package cmdtest_matchers

import (
	"fmt"
	"time"

	"github.com/vito/cmdtest"
)

func SayWithTimeout(pattern string, timeout time.Duration) *SayWithTimeoutMatcher {
	return &SayWithTimeoutMatcher{pattern, timeout}
}

type SayWithTimeoutMatcher struct {
	Pattern string
	Timeout time.Duration
}

func (m *SayWithTimeoutMatcher) Match(out interface{}) (bool, string, error) {
	session, ok := out.(*cmdtest.Session)
	if !ok {
		return false, "", fmt.Errorf("Cannot expect output from %#v.", out)
	}

	err := session.ExpectOutputWithTimeout(m.Pattern, m.Timeout)
	if err != nil {
		return false, err.Error(), nil
	}

	return true, fmt.Sprintf("Expected to not see %#v\n", m.Pattern), nil
}
