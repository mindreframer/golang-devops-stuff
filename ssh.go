package tachyon

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

type SSH struct {
	Host   string
	Config string
	Debug  bool

	removeConfig bool
	sshCCOptions []string
	sshCSOptions []string
	controlPath  string

	persistent *exec.Cmd
}

func (s *SSH) CommandWithOptions(cmd string, args ...string) []string {
	sshArgs := []string{cmd}
	sshArgs = append(sshArgs, s.sshCCOptions...)

	if s.Config != "" {
		sshArgs = append(sshArgs, "-F", s.Config)
	}

	return append(sshArgs, args...)
}

func (s *SSH) RsyncCommand() string {
	sshArgs := []string{"ssh"}
	sshArgs = append(sshArgs, s.sshCCOptions...)

	if s.Config != "" {
		sshArgs = append(sshArgs, "-F", s.Config)
	}

	return strings.Join(sshArgs, " ")
}

func (s *SSH) SSHCommand(cmd string, args ...string) []string {
	sshArgs := []string{cmd}
	sshArgs = append(sshArgs, s.sshCCOptions...)

	if s.Config != "" {
		sshArgs = append(sshArgs, "-F", s.Config)
	}

	sshArgs = append(sshArgs, s.Host)

	return append(sshArgs, args...)
}

func NewSSH(host string) *SSH {
	s := &SSH{
		Host: host,
	}

	if strings.Index(host, ":vagrant") == 0 {
		var target string

		if host == ":vagrant" {
			target = "default"
		} else {
			parts := strings.Split(host, ":")
			target = parts[2]
		}

		s.ImportVagrant(target)
	}

	home, err := HomeDir()
	if err != nil {
		panic(err)
	}

	tachDir := home + "/.tachyon"

	if _, err := os.Stat(tachDir); err != nil {
		err = os.Mkdir(tachDir, 0755)
		if err != nil {
			panic(err)
		}
	}

	s.controlPath = fmt.Sprintf("%s/tachyon-cp-ssh-%d", tachDir, os.Getpid())

	s.sshCCOptions = []string{"-o", "StrictHostKeyChecking=no"}

	s.sshCSOptions = []string{
		"-o", "ControlMaster=yes",
		"-o", "ControlPersist=no",
		"-o", "ControlPath=" + s.controlPath,
	}

	return s
}

func (s *SSH) Cleanup() {
	if s.persistent != nil {
		s.persistent.Process.Kill()
		s.persistent.Wait()
	}

	if s.removeConfig {
		os.Remove(s.Config)
	}
}

func (s *SSH) ImportVagrant(target string) bool {
	s.Host = target
	s.removeConfig = true

	out, err := exec.Command("vagrant", "ssh-config", target).CombinedOutput()
	if err != nil {
		fmt.Printf("Unable to execute 'vagrant ssh-config': %s\n", err)
		return false
	}

	f, err := ioutil.TempFile("", "tachyon")
	if err != nil {
		fmt.Printf("Unable to make tempfile: %s\n", err)
		return false
	}

	_, err = f.Write(out)
	if err != nil {
		fmt.Printf("Unable to write to tempfile: %s\n", err)
		return false
	}

	f.Close()

	s.Config = f.Name()

	return true
}

func (s *SSH) Start() error {
	s.sshCCOptions = append(s.sshCCOptions, "-S", s.controlPath)

	sshArgs := s.sshCSOptions

	if s.Config != "" {
		sshArgs = append(sshArgs, "-F", s.Config)
	}

	sshArgs = append(sshArgs, "-N", s.Host)

	c := exec.Command("ssh", sshArgs...)

	err := c.Start()
	if err == nil {
		s.persistent = c
	}

	return err
}

func (s *SSH) Command(args ...string) *exec.Cmd {
	args = s.SSHCommand("ssh", args...)
	return exec.Command(args[0], args[1:]...)
}

func (s *SSH) Run(args ...string) error {
	c := s.Command(args...)

	if s.Debug {
		fmt.Fprintf(os.Stderr, "Run: %#v\n", c.Args)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
	}

	return c.Run()
}

func (s *SSH) RunAndCapture(args ...string) ([]byte, error) {
	c := s.Command(args...)

	if s.Debug {
		fmt.Fprintf(os.Stderr, "Run: %#v\n", c.Args)
	}

	out, err := c.CombinedOutput()

	if s.Debug {
		fmt.Fprintf(os.Stderr, "Output:\n%s\n", string(out))
	}

	return out, err
}

func (s *SSH) RunAndShow(args ...string) error {
	c := s.Command(args...)

	if s.Debug {
		fmt.Fprintf(os.Stderr, "Run: %#v\n", c.Args)
	}

	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	return c.Run()
}

func (s *SSH) CopyToHost(src, dest string) error {
	args := s.CommandWithOptions("scp", src, s.Host+":"+dest)
	c := exec.Command(args[0], args[1:]...)

	if s.Debug {
		fmt.Fprintf(os.Stderr, "Run: %#v\n", c.Args)
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
	}

	return c.Run()
}
