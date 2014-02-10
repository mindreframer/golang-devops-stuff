package fake_command_runner_matchers

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/pivotal-cf-experimental/garden/command_runner/fake_command_runner"
)

func HaveSignalled(spec fake_command_runner.CommandSpec, signal os.Signal) *HaveSignalledMatcher {
	return &HaveSignalledMatcher{spec, signal}
}

type HaveSignalledMatcher struct {
	Spec   fake_command_runner.CommandSpec
	Signal os.Signal
}

func (m *HaveSignalledMatcher) Match(actual interface{}) (bool, string, error) {
	runner, ok := actual.(*fake_command_runner.FakeCommandRunner)
	if !ok {
		return false, "", fmt.Errorf("Not a fake command runner: %#v.", actual)
	}

	signalled := runner.SignalledCommands()

	matched := false
	for cmd, signal := range signalled {
		if m.Spec.Matches(cmd) {
			matched = signal == m.Signal
			break
		}
	}

	actuallySignalled := []*exec.Cmd{}
	for cmd, _ := range signalled {
		actuallySignalled = append(actuallySignalled, cmd)
	}

	if matched {
		return true, fmt.Sprintf("Expected to not signal %s to the following commands:%s", m.Signal, prettySpec(m.Spec)), nil
	} else {
		return false, fmt.Sprintf("Expected to signal %s to:%s\n\nActually signalled:%s", m.Signal, prettySpec(m.Spec), prettyCommands(actuallySignalled)), nil
	}
}
