package fake_command_runner_matchers

import (
	"fmt"

	"github.com/pivotal-cf-experimental/garden/command_runner/fake_command_runner"
)

func HaveKilled(spec fake_command_runner.CommandSpec) *HaveKilledMatcher {
	return &HaveKilledMatcher{spec}
}

type HaveKilledMatcher struct {
	Spec fake_command_runner.CommandSpec
}

func (m *HaveKilledMatcher) Match(actual interface{}) (bool, string, error) {
	runner, ok := actual.(*fake_command_runner.FakeCommandRunner)
	if !ok {
		return false, "", fmt.Errorf("Not a fake command runner: %#v.", actual)
	}

	killed := runner.KilledCommands()

	matched := false
	for _, cmd := range killed {
		if m.Spec.Matches(cmd) {
			matched = true
			break
		}
	}

	if matched {
		return true, fmt.Sprintf("Expected to not kill the following commands:%s", prettySpec(m.Spec)), nil
	} else {
		return false, fmt.Sprintf("Expected to kill:%s\n\nActually killed:%s", prettySpec(m.Spec), prettyCommands(killed)), nil
	}
}
