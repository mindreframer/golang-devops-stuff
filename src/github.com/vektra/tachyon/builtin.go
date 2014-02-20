package tachyon

import (
	"bufio"
	"github.com/flynn/go-shlex"
	"io"
	"os"
	"os/exec"
)

func setupNiceIO(c *exec.Cmd) error {
	stdout, err := c.StdoutPipe()

	if err != nil {
		return err
	}

	go func() {
		defer stdout.Close()

		prefix := []byte(`| `)
		buf := bufio.NewReader(stdout)

		for {
			line, err := buf.ReadSlice('\n')

			if err != nil {
				break
			}

			os.Stdout.Write(prefix)
			os.Stdout.Write(line)
		}
	}()

	stderr, err := c.StderrPipe()

	if err != nil {
		return err
	}

	go func() {
		defer stderr.Close()

		prefix := []byte(`| `)
		buf := bufio.NewReader(stderr)

		for {
			line, err := buf.ReadSlice('\n')

			if err != nil {
				break
			}

			os.Stdout.Write(prefix)
			os.Stdout.Write(line)
		}
	}()

	return nil
}

type CommandCmd struct{}

func (cmd *CommandCmd) Run(pe *PlayEnv, args string) error {
	parts, err := shlex.Split(args)

	if err != nil {
		return err
	}

	c := exec.Command(parts[0], parts[1:]...)
	if pe.config.ShowCommandOutput {
		setupNiceIO(c)
	}

	return c.Run()
}

type ShellCmd struct{}

func (cmd *ShellCmd) Run(pe *PlayEnv, args string) error {
	c := exec.Command("sh", "-c", args)

	if pe.config.ShowCommandOutput {
		setupNiceIO(c)
	}

	return c.Run()
}

type CopyCmd struct {
	Src  string `tachyon:"src,required"`
	Dest string `tachyon:"dest,required"`
}

func (cmd *CopyCmd) Run(pe *PlayEnv, args string) error {
	input, err := os.Open(cmd.Src)

	if err != nil {
		return err
	}

	defer input.Close()

	output, err := os.OpenFile(cmd.Dest, os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return err
	}

	defer output.Close()

	_, err = io.Copy(output, input)

	return err
}

func init() {
	RegisterCommand("command", &CommandCmd{})
	RegisterCommand("shell", &ShellCmd{})
	RegisterCommand("copy", &CopyCmd{})
}
