package command_runner_test

import (
	"os/exec"

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
		Expect(err).ToNot(HaveOccured())

		Expect(cmd.ProcessState).ToNot(BeNil())
	})

	Context("when the command fails", func() {
		It("returns an error", func() {
			runner := command_runner.New(false)

			err := runner.Run(&exec.Cmd{
				Path: "/bin/bash",
				Args: []string{"-c", "exit 1"},
			})

			Expect(err).To(HaveOccured())
		})
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
		Expect(err).ToNot(HaveOccured())

		Expect(cmd.ProcessState).To(BeNil())

		in.Write([]byte("hello\n"))

		cmd.Wait()

		Expect(cmd.ProcessState).ToNot(BeNil())
	})
})

var _ = Describe("Waiting on commands", func() {
	It("blocks on the command's completion", func() {
		runner := command_runner.New(false)

		cmd := &exec.Cmd{Path: "bash", Args: []string{"-c", "sleep 0.1"}}
		Expect(cmd.ProcessState).To(BeNil())

		err := runner.Start(cmd)
		Expect(err).ToNot(HaveOccured())

		Expect(cmd.ProcessState).To(BeNil())

		err = runner.Wait(cmd)
		Expect(err).ToNot(HaveOccured())

		Expect(cmd.ProcessState).ToNot(BeNil())
	})
})

var _ = Describe("Killing commands", func() {
	It("terminates the command's process", func() {
		runner := command_runner.New(false)

		cmd := &exec.Cmd{Path: "bash", Args: []string{"-c", "read foo"}}
		Expect(cmd.ProcessState).To(BeNil())

		err := runner.Start(cmd)
		Expect(err).ToNot(HaveOccured())

		Expect(cmd.ProcessState).To(BeNil())

		err = runner.Kill(cmd)
		Expect(err).ToNot(HaveOccured())

		err = cmd.Wait()
		Expect(err).To(HaveOccured())

		Expect(cmd.ProcessState).ToNot(BeNil())
	})

	Context("when the command is not running", func() {
		It("returns an error", func() {
			runner := command_runner.New(false)

			cmd := &exec.Cmd{Path: "bash", Args: []string{"-c", "read foo"}}
			Expect(cmd.ProcessState).To(BeNil())

			err := runner.Kill(cmd)
			Expect(err).To(HaveOccured())
		})
	})
})

var _ = Describe("Getting the root path of the host", func() {
	It("returns '/'", func() {
		runner := command_runner.New(false)
		Expect(runner.ServerRoot()).To(Equal("/"))
	})
})
