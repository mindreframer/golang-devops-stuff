package cmdtest_matchers

import (
	"fmt"

	"github.com/vito/cmdtest"
)

func SayBranches(branches ...cmdtest.ExpectBranch) *SayBranchesMatcher {
	return &SayBranchesMatcher{branches}
}

type SayBranchesMatcher struct {
	Branches []cmdtest.ExpectBranch
}

func (m *SayBranchesMatcher) Match(out interface{}) (bool, string, error) {
	session, ok := out.(*cmdtest.Session)
	if !ok {
		return false, "", fmt.Errorf("Cannot expect output from %#v.", out)
	}

	err := session.ExpectOutputBranches(m.Branches...)
	if err != nil {
		return false, err.Error(), nil
	}

	return true, "Expected to not see any of the branches.\n", nil
}
