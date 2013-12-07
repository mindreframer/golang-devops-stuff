package remote_command_runner

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/vito/garden/command_runner"
)

type RemoteCommandRunner struct {
	username string
	address  string
	port     uint32

	serverRoot string

	runner command_runner.CommandRunner
}

func New(username, address string, port uint32, serverRoot string, runner command_runner.CommandRunner) *RemoteCommandRunner {
	return &RemoteCommandRunner{
		username: username,
		address:  address,
		port:     port,

		serverRoot: serverRoot,

		runner: runner,
	}
}

func (r *RemoteCommandRunner) Run(cmd *exec.Cmd) error {
	return r.runner.Run(r.wrap(cmd))
}

func (r *RemoteCommandRunner) Start(cmd *exec.Cmd) error {
	return r.runner.Start(r.wrap(cmd))
}

func (r *RemoteCommandRunner) Wait(cmd *exec.Cmd) error {
	return r.runner.Wait(cmd)
}

func (r *RemoteCommandRunner) Kill(cmd *exec.Cmd) error {
	return r.runner.Kill(cmd)
}

func (r *RemoteCommandRunner) ServerRoot() string {
	return r.serverRoot
}

func (r *RemoteCommandRunner) wrap(cmd *exec.Cmd) *exec.Cmd {
	cmd.Args = []string{
		"-l", r.username,
		"-p", fmt.Sprintf("%d", r.port),
		r.address,
		r.buildCommandString(cmd.Env, cmd.Path, cmd.Args),
	}

	cmd.Path = "ssh"

	cmd.Env = []string{}

	return cmd
}

func (r *RemoteCommandRunner) buildCommandString(env []string, path string, args []string) string {
	cmd := []string{}

	cmd = append(cmd, env...)
	cmd = append(cmd, path)

	for _, arg := range args {
		cmd = append(cmd, r.quoteArg(arg))
	}

	return strings.Join(cmd, " ")
}

func (r *RemoteCommandRunner) quoteArg(arg string) string {
	// lol
	return "'" + strings.Replace(arg, `'`, `'"'"'`, -1) + "'"
}
