package fake_command_runner_matchers

import (
	"fmt"

	"github.com/vito/garden/command_runner/fake_command_runner"
)

func HaveExecutedSerially(specs ...fake_command_runner.CommandSpec) *HaveExecutedSeriallyMatcher {
	return &HaveExecutedSeriallyMatcher{specs}
}

type HaveExecutedSeriallyMatcher struct {
	Specs []fake_command_runner.CommandSpec
}

func (m *HaveExecutedSeriallyMatcher) Match(actual interface{}) (bool, string, error) {
	runner, ok := actual.(*fake_command_runner.FakeCommandRunner)
	if !ok {
		return false, "", fmt.Errorf("Not a fake command runner: %#v.", actual)
	}

	executed := runner.ExecutedCommands()

	matched := false
	startSearch := 0

	for _, spec := range m.Specs {
		matched = false

		for i := startSearch; i < len(executed); i++ {
			startSearch++

			if !spec.Matches(executed[i]) {
				continue
			}

			matched = true

			break
		}

		if !matched {
			break
		}
	}

	if matched {
		return true, fmt.Sprintf("Expected to not execute the following commands:%s", prettySpecs(m.Specs)), nil
	} else {
		return false, fmt.Sprintf("Expected to execute:%s\n\nActually executed:%s", prettySpecs(m.Specs), prettyCommands(executed)), nil
	}
}
