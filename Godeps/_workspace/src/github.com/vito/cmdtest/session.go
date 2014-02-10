package cmdtest

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type Session struct {
	Cmd *exec.Cmd

	Stdin io.WriteCloser

	stdout *Expector
	stderr *Expector

	exited chan int
}

type OutputWrapper func(io.Writer) io.Writer

func Start(cmd *exec.Cmd) (*Session, error) {
	return StartWrapped(cmd, noopWrapper, noopWrapper)
}

func StartWrapped(cmd *exec.Cmd, outWrapper OutputWrapper, errWrapper OutputWrapper) (*Session, error) {
	stdinOut, stdinIn, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	stdoutOut, stdoutIn, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	stderrOut, stderrIn, err := os.Pipe()
	if err != nil {
		return nil, err
	}

	cmd.Stdin = stdinOut
	cmd.Stdout = outWrapper(stdoutIn)
	cmd.Stderr = errWrapper(stderrIn)

	outExpector := NewExpector(stdoutOut, 0)
	errExpector := NewExpector(stderrOut, 0)

	err = cmd.Start()
	if err != nil {
		return nil, err
	}

	exited := make(chan int, 1)

	go func() {
		cmd.Wait()
		exited <- cmd.ProcessState.Sys().(syscall.WaitStatus).ExitStatus()

		stdoutIn.Close()
		stderrIn.Close()
	}()

	return &Session{
		Cmd: cmd,

		Stdin: stdinIn,

		stdout: outExpector,
		stderr: errExpector,

		exited: exited,
	}, nil
}

func (s Session) ExpectOutput(pattern string) error {
	return s.stdout.Expect(pattern)
}

func (s Session) ExpectOutputBranches(branches ...ExpectBranch) error {
	return s.stdout.ExpectBranches(branches...)
}

func (s Session) ExpectOutputWithTimeout(pattern string, timeout time.Duration) error {
	return s.stdout.ExpectWithTimeout(pattern, timeout)
}

func (s Session) ExpectError(pattern string) error {
	return s.stderr.Expect(pattern)
}

func (s Session) ExpectErrorWithTimeout(pattern string, timeout time.Duration) error {
	return s.stderr.ExpectWithTimeout(pattern, timeout)
}

func (s Session) Wait(timeout time.Duration) (int, error) {
	select {
	case status := <-s.exited:
		return status, nil
	case <-time.After(timeout):
		return -1, fmt.Errorf("command did not exit")
	}
}

func (s Session) FullOutput() []byte {
	return s.stdout.FullOutput()
}

func (s Session) FullErrorOutput() []byte {
	return s.stderr.FullOutput()
}

func noopWrapper(out io.Writer) io.Writer {
	return out
}
