package job_tracker_test

import (
	"bytes"
	"errors"
	"fmt"
	"os/exec"
	"path"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vito/garden/backend/linux_backend/job_tracker"
	"github.com/vito/garden/command_runner/fake_command_runner"
	. "github.com/vito/garden/command_runner/fake_command_runner/matchers"
)

var fakeRunner *fake_command_runner.FakeCommandRunner
var jobTracker *job_tracker.JobTracker

func binPath(bin string) string {
	return path.Join("/depot/some-id", "bin", bin)
}

func setupSuccessfulSpawn() {
	fakeRunner.WhenRunning(
		fake_command_runner.CommandSpec{
			Path: binPath("iomux-spawn"),
		},
		func(cmd *exec.Cmd) error {
			cmd.Stdout.Write([]byte("ready\n"))
			cmd.Stdout.Write([]byte("active\n"))
			return nil
		},
	)
}

var _ = Describe("Spawning jobs", func() {
	BeforeEach(func() {
		fakeRunner = fake_command_runner.New()
		jobTracker = job_tracker.New("/depot/some-id", fakeRunner)
	})

	It("runs the command asynchronously via iomux-spawn", func() {
		cmd := &exec.Cmd{Path: "/bin/bash"}

		cmd.Stdin = bytes.NewBufferString("echo hi")

		setupSuccessfulSpawn()

		jobID, _ := jobTracker.Spawn(cmd, false)

		Eventually(fakeRunner).Should(HaveStartedExecuting(
			fake_command_runner.CommandSpec{
				Path: binPath("iomux-spawn"),
				Args: []string{
					fmt.Sprintf("/depot/some-id/jobs/%d", jobID),
					"/bin/bash",
				},
				Stdin: "echo hi",
			},
		))
	})

	It("initiates a link to the job after spawn is ready", func(done Done) {
		fakeRunner.WhenRunning(
			fake_command_runner.CommandSpec{
				Path: binPath("iomux-spawn"),
			}, func(cmd *exec.Cmd) error {
				go func() {
					time.Sleep(100 * time.Millisecond)

					Expect(fakeRunner).ToNot(HaveExecutedSerially(
						fake_command_runner.CommandSpec{
							Path: binPath("iomux-link"),
						},
					), "Executed iomux-link too early!")

					if cmd.Stdout != nil {
						cmd.Stdout.Write([]byte("xxx\n"))
					}

					Eventually(fakeRunner).Should(HaveExecutedSerially(
						fake_command_runner.CommandSpec{
							Path: binPath("iomux-link"),
						},
					))

					close(done)
				}()

				return nil
			},
		)

		jobTracker.Spawn(exec.Command("xxx"), false)
	}, 10.0)

	It("returns a unique job ID", func() {
		setupSuccessfulSpawn()

		jobID1, _ := jobTracker.Spawn(exec.Command("xxx"), false)
		jobID2, _ := jobTracker.Spawn(exec.Command("xxx"), false)
		Expect(jobID1).ToNot(Equal(jobID2))
	})

	It("creates the job's working directory", func() {
		setupSuccessfulSpawn()

		jobID, _ := jobTracker.Spawn(exec.Command("xxx"), false)

		Expect(fakeRunner).To(HaveExecutedSerially(
			fake_command_runner.CommandSpec{
				Path: "mkdir",
				Args: []string{
					"-p",
					fmt.Sprintf("/depot/some-id/jobs/%d", jobID),
				},
			},
		))
	})

	Context("when spawning fails", func() {
		disaster := errors.New("oh no!")

		BeforeEach(func() {
			fakeRunner.WhenRunning(
				fake_command_runner.CommandSpec{
					Path: binPath("iomux-spawn"),
				}, func(*exec.Cmd) error {
					return disaster
				},
			)
		})

		It("returns the error", func() {
			_, err := jobTracker.Spawn(exec.Command("xxx"), false)
			Expect(err).To(Equal(disaster))
		})
	})
})

var _ = Describe("Linking to jobs", func() {
	BeforeEach(func() {
		fakeRunner = fake_command_runner.New()
		jobTracker = job_tracker.New("/depot/some-id", fakeRunner)

		fakeRunner.WhenRunning(
			fake_command_runner.CommandSpec{
				Path: binPath("iomux-link"),
			},
			func(cmd *exec.Cmd) error {
				cmd.Stdout.Write([]byte("hi out\n"))
				cmd.Stderr.Write([]byte("hi err\n"))

				dummyCmd := exec.Command("/bin/bash", "-c", "exit 42")
				dummyCmd.Run()

				cmd.ProcessState = dummyCmd.ProcessState

				return nil
			},
		)
	})

	It("returns their stdout, stderr, and exit status", func() {
		setupSuccessfulSpawn()

		jobID, _ := jobTracker.Spawn(exec.Command("xxx"), false)

		exitStatus, stdout, stderr, err := jobTracker.Link(jobID)
		Expect(err).ToNot(HaveOccured())
		Expect(exitStatus).To(Equal(uint32(42)))
		Expect(stdout).To(Equal([]byte("hi out\n")))
		Expect(stderr).To(Equal([]byte("hi err\n")))
	})

	Context("when the output is discarded", func() {
		It("returns the exit status but no stdout/stderr", func() {
			setupSuccessfulSpawn()

			jobID, _ := jobTracker.Spawn(exec.Command("xxx"), true)

			exitStatus, stdout, stderr, err := jobTracker.Link(jobID)
			Expect(err).ToNot(HaveOccured())
			Expect(exitStatus).To(Equal(uint32(42)))
			Expect(stdout).To(BeEmpty())
			Expect(stderr).To(BeEmpty())
		})
	})

	Context("when more than one link is made", func() {
		BeforeEach(func() {
			fakeRunner.WhenRunning(
				fake_command_runner.CommandSpec{
					Path: binPath("iomux-spawn"),
				},
				func(cmd *exec.Cmd) error {
					// give time for both goroutines to link
					time.Sleep(1000 * time.Millisecond)
					cmd.Stdout.Write([]byte("ready\n"))
					cmd.Stdout.Write([]byte("active\n"))
					return nil
				},
			)

			fakeRunner.WhenRunning(
				fake_command_runner.CommandSpec{
					Path: binPath("iomux-link"),
				},
				func(cmd *exec.Cmd) error {
					cmd.Stdout.Write([]byte("hi out\n"))
					cmd.Stderr.Write([]byte("hi err\n"))

					dummyCmd := exec.Command("/bin/bash", "-c", "exit 42")
					dummyCmd.Run()

					cmd.ProcessState = dummyCmd.ProcessState

					return nil
				},
			)
		})

		// TODO: this test is racey
		It("returns to both", func(done Done) {
			jobID, _ := jobTracker.Spawn(exec.Command("xxx"), false)

			finishedLink := make(chan bool)

			go func() {
				exitStatus, stdout, stderr, err := jobTracker.Link(jobID)
				Expect(err).ToNot(HaveOccured())
				Expect(exitStatus).To(Equal(uint32(42)))
				Expect(string(stdout)).To(Equal("hi out\n"))
				Expect(string(stderr)).To(Equal("hi err\n"))

				finishedLink <- true
			}()

			go func() {
				exitStatus, stdout, stderr, err := jobTracker.Link(jobID)
				Expect(err).ToNot(HaveOccured())
				Expect(exitStatus).To(Equal(uint32(42)))
				Expect(string(stdout)).To(Equal("hi out\n"))
				Expect(string(stderr)).To(Equal("hi err\n"))

				finishedLink <- true
			}()

			<-finishedLink
			<-finishedLink

			close(done)
		}, 10.0)
	})
})

var _ = Describe("Streaming jobs", func() {
	BeforeEach(func() {
		fakeRunner = fake_command_runner.New()
		jobTracker = job_tracker.New("/depot/some-id", fakeRunner)

		fakeRunner.WhenRunning(
			fake_command_runner.CommandSpec{
				Path: binPath("iomux-link"),
			},
			func(cmd *exec.Cmd) error {
				time.Sleep(100 * time.Millisecond)

				cmd.Stdout.Write([]byte("hi out\n"))

				time.Sleep(100 * time.Millisecond)

				cmd.Stderr.Write([]byte("hi err\n"))

				time.Sleep(100 * time.Millisecond)

				dummyCmd := exec.Command("/bin/bash", "-c", "exit 42")
				dummyCmd.Run()

				cmd.ProcessState = dummyCmd.ProcessState

				return nil
			},
		)
	})

	It("streams their stdout and stderr into the channel", func(done Done) {
		setupSuccessfulSpawn()

		jobID, _ := jobTracker.Spawn(exec.Command("xxx"), false)

		jobStreamChannel, err := jobTracker.Stream(jobID)
		Expect(err).ToNot(HaveOccured())

		chunk1 := <-jobStreamChannel
		Expect(chunk1.Name).To(Equal("stdout"))
		Expect(string(chunk1.Data)).To(Equal("hi out\n"))
		Expect(chunk1.ExitStatus).To(BeNil())
		Expect(chunk1.Info).To(BeNil())

		chunk2 := <-jobStreamChannel
		Expect(chunk2.Name).To(Equal("stderr"))
		Expect(string(chunk2.Data)).To(Equal("hi err\n"))
		Expect(chunk2.ExitStatus).To(BeNil())
		Expect(chunk2.Info).To(BeNil())

		close(done)
	}, 5.0)

	Context("when the job completes", func() {
		It("yields the exit status and closes the channel", func(done Done) {
			setupSuccessfulSpawn()

			jobID, _ := jobTracker.Spawn(exec.Command("xxx"), false)

			jobStreamChannel, err := jobTracker.Stream(jobID)
			Expect(err).ToNot(HaveOccured())

			<-jobStreamChannel
			<-jobStreamChannel

			chunk3 := <-jobStreamChannel
			Expect(chunk3.Name).To(Equal(""))
			Expect(string(chunk3.Data)).To(Equal(""))
			Expect(chunk3.ExitStatus).ToNot(BeNil())
			Expect(*chunk3.ExitStatus).To(Equal(uint32(42)))
			//Expect(chunk3.Info).ToNot(BeNil())

			_, ok := <-jobStreamChannel
			Expect(ok).To(BeFalse(), "channel is not closed")

			close(done)
		}, 5.0)
	})
})

var _ = Describe("Listing active jobs", func() {
	BeforeEach(func() {
		fakeRunner = fake_command_runner.New()
		jobTracker = job_tracker.New("/depot/some-id", fakeRunner)
	})

	It("includes running job IDs", func() {
		setupSuccessfulSpawn()

		running := make(chan []uint32, 2)

		fakeRunner.WhenRunning(
			fake_command_runner.CommandSpec{
				Path: binPath("iomux-link"),
			},
			func(cmd *exec.Cmd) error {
				running <- jobTracker.ActiveJobs()
				return nil
			},
		)

		jobID1, err := jobTracker.Spawn(exec.Command("xxx"), false)
		Expect(err).ToNot(HaveOccured())

		jobID2, err := jobTracker.Spawn(exec.Command("xxx"), false)
		Expect(err).ToNot(HaveOccured())

		totalRunning := append(<-running, <-running...)

		Expect(totalRunning).To(ContainElement(jobID1))
		Expect(totalRunning).To(ContainElement(jobID2))
	})
})
