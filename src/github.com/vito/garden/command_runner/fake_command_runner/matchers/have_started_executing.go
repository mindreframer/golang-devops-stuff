package fake_command_runner_matchers

import (
	"fmt"

	"github.com/pivotal-cf-experimental/garden/command_runner/fake_command_runner"
)

func HaveStartedExecuting(spec fake_command_runner.CommandSpec) *HaveStartedExecutingMatcher {
	return &HaveStartedExecutingMatcher{spec}
}

type HaveStartedExecutingMatcher struct {
	Spec fake_command_runner.CommandSpec
}

func (m *HaveStartedExecutingMatcher) Match(actual interface{}) (bool, string, error) {
	runner, ok := actual.(*fake_command_runner.FakeCommandRunner)
	if !ok {
		return false, "", fmt.Errorf("Not a fake command runner: %#v.", actual)
	}

	started := runner.StartedCommands()

	matched := false
	for _, cmd := range started {
		if m.Spec.Matches(cmd) {
			matched = true
			break
		}
	}

	if matched {
		return true, fmt.Sprintf("Expected to not start the following commands:%s", prettySpec(m.Spec)), nil
	} else {
		return false, fmt.Sprintf("Expected to start:%s\n\nActually started:%s", prettySpec(m.Spec), prettyCommands(started)), nil
	}
}
