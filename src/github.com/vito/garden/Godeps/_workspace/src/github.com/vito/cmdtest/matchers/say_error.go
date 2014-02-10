package cmdtest_matchers

import (
	"fmt"

	"github.com/vito/cmdtest"
)

func SayError(pattern string) *SayErrorMatcher {
	return &SayErrorMatcher{pattern}
}

type SayErrorMatcher struct {
	Pattern string
}

func (m *SayErrorMatcher) Match(out interface{}) (bool, string, error) {
	session, ok := out.(*cmdtest.Session)
	if !ok {
		return false, "", fmt.Errorf("Cannot expect output from %#v.", out)
	}

	err := session.ExpectError(m.Pattern)
	if err != nil {
		return false, err.Error(), nil
	}

	return true, fmt.Sprintf("Expected to not see %#v\n", m.Pattern), nil
}
