package fake_command_runner

import (
	"os/exec"
	"reflect"

	"sync"
)

type FakeCommandRunner struct {
	ServerRootPath string

	executedCommands []*exec.Cmd
	startedCommands  []*exec.Cmd
	waitedCommands   []*exec.Cmd
	killedCommands   []*exec.Cmd

	commandCallbacks map[*CommandSpec]func(*exec.Cmd) error
	waitingCallbacks map[*CommandSpec]func(*exec.Cmd) error

	sync.RWMutex
}

type CommandSpec struct {
	Path  string
	Args  []string
	Env   []string
	Stdin string
}

func (s CommandSpec) Matches(cmd *exec.Cmd) bool {
	if s.Path != "" && s.Path != cmd.Path {
		return false
	}

	if len(s.Args) > 0 && !reflect.DeepEqual(s.Args, cmd.Args) {
		return false
	}

	if len(s.Env) > 0 && !reflect.DeepEqual(s.Env, cmd.Env) {
		return false
	}

	if s.Stdin != "" {
		if cmd.Stdin == nil {
			return false
		}

		in := make([]byte, len(s.Stdin))
		_, err := cmd.Stdin.Read(in)
		if err != nil {
			return false
		}

		if string(in) != s.Stdin {
			return false
		}
	}

	return true
}

func New() *FakeCommandRunner {
	return &FakeCommandRunner{
		commandCallbacks: make(map[*CommandSpec]func(*exec.Cmd) error),
		waitingCallbacks: make(map[*CommandSpec]func(*exec.Cmd) error),
	}
}

func (r *FakeCommandRunner) Run(cmd *exec.Cmd) error {
	r.RLock()
	callbacks := r.commandCallbacks
	r.RUnlock()

	r.Lock()
	r.executedCommands = append(r.executedCommands, cmd)
	r.Unlock()

	for spec, callback := range callbacks {
		if spec.Matches(cmd) {
			return callback(cmd)
		}
	}

	return nil
}

func (r *FakeCommandRunner) Start(cmd *exec.Cmd) error {
	r.RLock()
	callbacks := r.commandCallbacks
	r.RUnlock()

	r.Lock()
	r.startedCommands = append(r.startedCommands, cmd)
	r.Unlock()

	for spec, callback := range callbacks {
		if spec.Matches(cmd) {
			return callback(cmd)
		}
	}

	return nil
}

func (r *FakeCommandRunner) Wait(cmd *exec.Cmd) error {
	r.RLock()
	callbacks := r.waitingCallbacks
	r.RUnlock()

	r.Lock()
	r.waitedCommands = append(r.waitedCommands, cmd)
	r.Unlock()

	for spec, callback := range callbacks {
		if spec.Matches(cmd) {
			return callback(cmd)
		}
	}

	return nil
}

func (r *FakeCommandRunner) Kill(cmd *exec.Cmd) error {
	r.Lock()
	defer r.Unlock()

	r.killedCommands = append(r.waitedCommands, cmd)

	return nil
}

func (r *FakeCommandRunner) ServerRoot() string {
	return r.ServerRootPath
}

func (r *FakeCommandRunner) WhenRunning(spec CommandSpec, callback func(*exec.Cmd) error) {
	r.Lock()
	defer r.Unlock()

	r.commandCallbacks[&spec] = callback
}

func (r *FakeCommandRunner) WhenWaitingFor(spec CommandSpec, callback func(*exec.Cmd) error) {
	r.Lock()
	defer r.Unlock()

	r.waitingCallbacks[&spec] = callback
}

func (r *FakeCommandRunner) ExecutedCommands() []*exec.Cmd {
	r.RLock()
	defer r.RUnlock()

	return r.executedCommands
}

func (r *FakeCommandRunner) StartedCommands() []*exec.Cmd {
	r.RLock()
	defer r.RUnlock()

	return r.startedCommands
}

func (r *FakeCommandRunner) KilledCommands() []*exec.Cmd {
	r.RLock()
	defer r.RUnlock()

	return r.killedCommands
}
