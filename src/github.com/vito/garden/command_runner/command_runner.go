package command_runner

import (
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"syscall"
)

type CommandRunner interface {
	Run(*exec.Cmd) error
	Start(*exec.Cmd) error
	Wait(*exec.Cmd) error
	Kill(*exec.Cmd) error
	Signal(*exec.Cmd, os.Signal) error
	ServerRoot() string
}

type RealCommandRunner struct {
	debug bool
}

type CommandNotRunningError struct {
	cmd *exec.Cmd
}

func (e CommandNotRunningError) Error() string {
	return fmt.Sprintf("command is not running: %#v", e.cmd)
}

func New(debug bool) *RealCommandRunner {
	return &RealCommandRunner{debug}
}

func (r *RealCommandRunner) Run(cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
	} else {
		cmd.SysProcAttr.Setpgid = true
	}

	if r.debug {
		log.Printf("\x1b[40;36mexecuting: %s\x1b[0m\n", prettyCommand(cmd))
		r.tee(cmd)
	}

	err := r.resolve(cmd).Run()

	if r.debug {
		if err != nil {
			log.Printf("\x1b[40;31mcommand failed (%s): %s\x1b[0m\n", prettyCommand(cmd), err)
		} else {
			log.Printf("\x1b[40;32mcommand succeeded (%s)\x1b[0m\n", prettyCommand(cmd))
		}
	}

	return err
}

func (r *RealCommandRunner) Start(cmd *exec.Cmd) error {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{
			Setpgid: true,
		}
	} else {
		cmd.SysProcAttr.Setpgid = true
	}

	if r.debug {
		log.Printf("\x1b[40;36mspawning: %s\x1b[0m\n", prettyCommand(cmd))
		r.tee(cmd)
	}

	err := r.resolve(cmd).Start()

	if r.debug {
		if err != nil {
			log.Printf("\x1b[40;31mspawning failed: %s\x1b[0m\n", err)
		} else {
			log.Printf("\x1b[40;32mspawning succeeded\x1b[0m\n")
		}
	}

	return err
}

func (r *RealCommandRunner) Wait(cmd *exec.Cmd) error {
	return cmd.Wait()
}

func (r *RealCommandRunner) Kill(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return CommandNotRunningError{cmd}
	}

	return cmd.Process.Kill()
}

func (r *RealCommandRunner) Signal(cmd *exec.Cmd, signal os.Signal) error {
	if cmd.Process == nil {
		return CommandNotRunningError{cmd}
	}

	return cmd.Process.Signal(signal)
}

func (r *RealCommandRunner) ServerRoot() string {
	return "/"
}

func (r *RealCommandRunner) tee(cmd *exec.Cmd) {
	if cmd.Stderr == nil {
		cmd.Stderr = os.Stderr
	} else if cmd.Stderr != nil {
		cmd.Stderr = io.MultiWriter(cmd.Stderr, os.Stderr)
	}

	if cmd.Stdout == nil {
		cmd.Stdout = os.Stderr

	} else if cmd.Stdout != nil {
		cmd.Stdout = io.MultiWriter(cmd.Stdout, os.Stderr)
	}
}

func (r *RealCommandRunner) resolve(cmd *exec.Cmd) *exec.Cmd {
	originalPath := cmd.Path

	path, err := exec.LookPath(cmd.Path)
	if err != nil {
		path = cmd.Path
	}

	cmd.Path = path

	cmd.Args = append([]string{originalPath}, cmd.Args...)

	return cmd
}

func prettyCommand(cmd *exec.Cmd) string {
	return fmt.Sprintf("%v %s %v", cmd.Env, cmd.Path, cmd.Args)
}
