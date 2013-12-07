package cgroups_manager_test

import (
	"errors"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vito/garden/backend/linux_backend/cgroups_manager"
	"github.com/vito/garden/command_runner/fake_command_runner"
	. "github.com/vito/garden/command_runner/fake_command_runner/matchers"
)

var _ = Describe("Container cgroups", func() {
	var fakeRunner *fake_command_runner.FakeCommandRunner
	var cgroupsManager *cgroups_manager.ContainerCgroupsManager

	BeforeEach(func() {
		fakeRunner = fake_command_runner.New()

		cgroupsManager = cgroups_manager.New(
			"/cgroup/path",
			"some-container-id",
			fakeRunner,
		)
	})

	Describe("setting", func() {
		It("writes the value to the name under the subsytem", func() {
			err := cgroupsManager.Set("memory", "memory.limit_in_bytes", "42")
			Expect(err).ToNot(HaveOccured())

			Expect(fakeRunner).To(HaveExecutedSerially(
				fake_command_runner.CommandSpec{
					Path: "bash",
					Args: []string{
						"-c",
						"echo '42' > /cgroup/path/memory/instance-some-container-id/memory.limit_in_bytes",
					},
				},
			))
		})

		Context("when setting the value fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeRunner.WhenRunning(fake_command_runner.CommandSpec{
					Path: "bash",
				}, func(*exec.Cmd) error {
					return disaster
				})
			})

			It("returns an error", func() {
				err := cgroupsManager.Set("memory", "memory.limit_in_bytes", "42")
				Expect(err).To(Equal(disaster))
			})
		})
	})

	Describe("getting", func() {
		It("reads the current value from the name under the subsystem", func() {
			fakeRunner.WhenRunning(
				fake_command_runner.CommandSpec{
					Path: "cat",
					Args: []string{
						"/cgroup/path/memory/instance-some-container-id/memory.limit_in_bytes",
					},
				},
				func(cmd *exec.Cmd) error {
					cmd.Stdout.Write([]byte("42\n"))
					return nil
				},
			)

			val, err := cgroupsManager.Get("memory", "memory.limit_in_bytes")
			Expect(err).ToNot(HaveOccured())

			Expect(val).To(Equal("42"))
		})
	})

	Describe("retrieving a subsystem path", func() {
		It("returns <path>/<subsytem>/instance-<container-id>", func() {
			Expect(cgroupsManager.SubsystemPath("memory")).To(Equal(
				"/cgroup/path/memory/instance-some-container-id",
			))
		})
	})
})
