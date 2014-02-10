package cmdtest_matchers

import (
	"fmt"

	"github.com/vito/cmdtest"
)

func Say(pattern string) *SayMatcher {
	return &SayMatcher{pattern}
}

type SayMatcher struct {
	Pattern string
}

func (m *SayMatcher) Match(out interface{}) (bool, string, error) {
	session, ok := out.(*cmdtest.Session)
	if !ok {
		return false, "", fmt.Errorf("Cannot expect output from %#v.", out)
	}

	err := session.ExpectOutput(m.Pattern)
	if err != nil {
		return false, err.Error(), nil
	}

	return true, fmt.Sprintf("Expected to not see %#v\n", m.Pattern), nil
}
