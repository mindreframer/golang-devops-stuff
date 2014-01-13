package command_runner_test

import (
	"os"
	"os/exec"
	"syscall"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vito/garden/command_runner"
)

var _ = Describe("Running commands", func() {
	It("runs the command and returns nil", func() {
		runner := command_runner.New(false)

		cmd := &exec.Cmd{Path: "ls"}
		Expect(cmd.ProcessState).To(BeNil())

		err := runner.Run(cmd)
		Expect(err).ToNot(HaveOccurred())

		Expect(cmd.ProcessState).ToNot(BeNil())
	})

	Context("when the command fails", func() {
		It("returns an error", func() {
			runner := command_runner.New(false)

			err := runner.Run(&exec.Cmd{
				Path: "/bin/bash",
				Args: []string{"-c", "exit 1"},
			})

			Expect(err).To(HaveOccurred())
		})
	})

	It("does not propagate signals to the child", func() {
		runner := command_runner.New(false)

		cmd := &exec.Cmd{
			Path: "/bin/bash",
			Args: []string{"-c", "exit 0"},
		}

		err := runner.Run(cmd)
		Expect(err).ToNot(HaveOccurred())

		Expect(cmd.SysProcAttr).ToNot(BeNil())
		Expect(cmd.SysProcAttr.Setpgid).To(BeTrue())
	})
})

var _ = Describe("Starting commands", func() {
	It("starts the command and does not block on it", func() {
		runner := command_runner.New(false)

		cmd := &exec.Cmd{Path: "bash", Args: []string{"-c", "read foo"}}
		Expect(cmd.ProcessState).To(BeNil())

		in, err := cmd.StdinPipe()
		Expect(err).To(BeNil())

		err = runner.Start(cmd)
		Expect(err).ToNot(HaveOccurred())

		Expect(cmd.ProcessState).To(BeNil())

		in.Write([]byte("hello\n"))

		cmd.Wait()

		Expect(cmd.ProcessState).ToNot(BeNil())
	})

	It("does not propagate signals to the child", func() {
		runner := command_runner.New(false)

		cmd := &exec.Cmd{
			Path: "/bin/bash",
			Args: []string{"-c", "exit 0"},
		}

		err := runner.Start(cmd)
		Expect(err).ToNot(HaveOccurred())

		Expect(cmd.SysProcAttr).ToNot(BeNil())
		Expect(cmd.SysProcAttr.Setpgid).To(BeTrue())
	})
})

var _ = Describe("Waiting on commands", func() {
	It("blocks on the command's completion", func() {
		runner := command_runner.New(false)

		cmd := &exec.Cmd{Path: "bash", Args: []string{"-c", "sleep 0.1"}}
		Expect(cmd.ProcessState).To(BeNil())

		err := runner.Start(cmd)
		Expect(err).ToNot(HaveOccurred())

		Expect(cmd.ProcessState).To(BeNil())

		err = runner.Wait(cmd)
		Expect(err).ToNot(HaveOccurred())

		Expect(cmd.ProcessState).ToNot(BeNil())
	})
})

var _ = Describe("Killing commands", func() {
	It("terminates the command's process", func() {
		runner := command_runner.New(false)

		cmd := &exec.Cmd{Path: "bash", Args: []string{"-c", "sleep 10"}}
		Expect(cmd.ProcessState).To(BeNil())

		err := runner.Start(cmd)
		Expect(err).ToNot(HaveOccurred())

		Expect(cmd.ProcessState).To(BeNil())

		err = runner.Kill(cmd)
		Expect(err).ToNot(HaveOccurred())

		err = cmd.Wait()
		Expect(err).To(HaveOccurred())

		Expect(cmd.ProcessState).ToNot(BeNil())
	})

	Context("when the command is not running", func() {
		It("returns an error", func() {
			runner := command_runner.New(false)

			cmd := &exec.Cmd{Path: "bash", Args: []string{"-c", "sleep 10"}}
			Expect(cmd.ProcessState).To(BeNil())

			err := runner.Kill(cmd)
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("Signalling commands", func() {
	It("sends the given signal to the process", func() {
		runner := command_runner.New(false)

		cmd := &exec.Cmd{Path: "bash", Args: []string{"-c", "sleep 10"}}
		Expect(cmd.ProcessState).To(BeNil())

		err := runner.Start(cmd)
		Expect(err).ToNot(HaveOccurred())

		Expect(cmd.ProcessState).To(BeNil())

		err = runner.Signal(cmd, os.Interrupt)
		Expect(err).ToNot(HaveOccurred())

		err = cmd.Wait()
		Expect(err).To(HaveOccurred())

		Expect(cmd.ProcessState.Sys().(syscall.WaitStatus).Signal()).To(Equal(os.Interrupt))
	})

	Context("when the command is not running", func() {
		It("returns an error", func() {
			runner := command_runner.New(false)

			cmd := &exec.Cmd{Path: "bash", Args: []string{"-c", "read foo"}}
			Expect(cmd.ProcessState).To(BeNil())

			err := runner.Signal(cmd, os.Interrupt)
			Expect(err).To(HaveOccurred())
		})
	})
})

var _ = Describe("Getting the root path of the host", func() {
	It("returns '/'", func() {
		runner := command_runner.New(false)
		Expect(runner.ServerRoot()).To(Equal("/"))
	})
})
