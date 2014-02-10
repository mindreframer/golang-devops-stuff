package linux_backend_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net"
	"os"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf-experimental/garden/backend"
	"github.com/pivotal-cf-experimental/garden/command_runner/fake_command_runner"
	. "github.com/pivotal-cf-experimental/garden/command_runner/fake_command_runner/matchers"
	"github.com/pivotal-cf-experimental/garden/linux_backend"
	"github.com/pivotal-cf-experimental/garden/linux_backend/bandwidth_manager/fake_bandwidth_manager"
	"github.com/pivotal-cf-experimental/garden/linux_backend/cgroups_manager/fake_cgroups_manager"
	"github.com/pivotal-cf-experimental/garden/linux_backend/network_pool"
	"github.com/pivotal-cf-experimental/garden/linux_backend/port_pool/fake_port_pool"
	"github.com/pivotal-cf-experimental/garden/linux_backend/quota_manager/fake_quota_manager"
)

var fakeCgroups *fake_cgroups_manager.FakeCgroupsManager
var fakeQuotaManager *fake_quota_manager.FakeQuotaManager
var fakeBandwidthManager *fake_bandwidth_manager.FakeBandwidthManager
var fakeRunner *fake_command_runner.FakeCommandRunner
var containerResources *linux_backend.Resources
var container *linux_backend.LinuxContainer
var fakePortPool *fake_port_pool.FakePortPool

var _ = Describe("Linux containers", func() {
	BeforeEach(func() {
		fakeRunner = fake_command_runner.New()

		fakeCgroups = fake_cgroups_manager.New("/cgroups", "some-id")

		fakeQuotaManager = fake_quota_manager.New()
		fakeBandwidthManager = fake_bandwidth_manager.New()

		_, ipNet, err := net.ParseCIDR("10.254.0.0/24")
		Expect(err).ToNot(HaveOccurred())

		fakePortPool = fake_port_pool.New(1000)

		networkPool := network_pool.New(ipNet)

		network, err := networkPool.Acquire()
		Expect(err).ToNot(HaveOccurred())

		containerResources = linux_backend.NewResources(
			1234,
			network,
			[]uint32{},
		)

		container = linux_backend.NewLinuxContainer(
			"some-id",
			"some-handle",
			"/depot/some-id",
			1*time.Second,
			containerResources,
			fakePortPool,
			fakeRunner,
			fakeCgroups,
			fakeQuotaManager,
			fakeBandwidthManager,
		)
	})

	setupSuccessfulSpawn := func() {
		fakeRunner.WhenRunning(
			fake_command_runner.CommandSpec{
				Path: "/depot/some-id/bin/iomux-spawn",
			},
			func(cmd *exec.Cmd) error {
				cmd.Stdout.Write([]byte("ready\n"))
				cmd.Stdout.Write([]byte("active\n"))
				return nil
			},
		)
	}

	Describe("Snapshotting", func() {
		It("writes a JSON ContainerSnapshot", func() {
			var err error

			err = container.Start()
			Expect(err).ToNot(HaveOccurred())

			memoryLimits := backend.MemoryLimits{
				LimitInBytes: 1,
			}

			diskLimits := backend.DiskLimits{
				BlockLimit: 1,
				Block:      2,
				BlockSoft:  3,
				BlockHard:  4,

				InodeLimit: 11,
				Inode:      12,
				InodeSoft:  13,
				InodeHard:  14,

				ByteLimit: 21,
				Byte:      22,
				ByteSoft:  23,
				ByteHard:  24,
			}

			bandwidthLimits := backend.BandwidthLimits{
				RateInBytesPerSecond:      1,
				BurstRateInBytesPerSecond: 2,
			}

			cpuLimits := backend.CPULimits{
				LimitInShares: 1,
			}

			err = container.LimitMemory(memoryLimits)
			Expect(err).ToNot(HaveOccurred())

			// oom exits immediately since it's faked out; should see event,
			// and it should show up in the snapshot
			Eventually(container.Events).Should(ContainElement("out of memory"))

			err = container.LimitDisk(diskLimits)
			Expect(err).ToNot(HaveOccurred())

			err = container.LimitBandwidth(bandwidthLimits)
			Expect(err).ToNot(HaveOccurred())

			err = container.LimitCPU(cpuLimits)
			Expect(err).ToNot(HaveOccurred())

			_, _, err = container.NetIn(1, 2)
			Expect(err).ToNot(HaveOccurred())

			_, _, err = container.NetIn(3, 4)
			Expect(err).ToNot(HaveOccurred())

			err = container.NetOut("network-a", 1)
			Expect(err).ToNot(HaveOccurred())

			err = container.NetOut("network-b", 2)
			Expect(err).ToNot(HaveOccurred())

			setupSuccessfulSpawn()

			_, err = container.Spawn(backend.JobSpec{
				DiscardOutput: true,
				AutoLink:      false,
			})
			Expect(err).ToNot(HaveOccurred())

			_, err = container.Spawn(backend.JobSpec{
				DiscardOutput: false,
				AutoLink:      false,
			})
			Expect(err).ToNot(HaveOccurred())

			out := new(bytes.Buffer)

			err = container.Snapshot(out)
			Expect(err).ToNot(HaveOccurred())

			var snapshot linux_backend.ContainerSnapshot

			err = json.NewDecoder(out).Decode(&snapshot)
			Expect(err).ToNot(HaveOccurred())

			Expect(snapshot.ID).To(Equal("some-id"))
			Expect(snapshot.Handle).To(Equal("some-handle"))

			Expect(snapshot.GraceTime).To(Equal(1 * time.Second))

			Expect(snapshot.State).To(Equal("stopped"))
			Expect(snapshot.Events).To(Equal([]string{"out of memory"}))

			Expect(snapshot.Limits).To(Equal(
				linux_backend.LimitsSnapshot{
					Memory:    &memoryLimits,
					Disk:      &diskLimits,
					Bandwidth: &bandwidthLimits,
					CPU:       &cpuLimits,
				},
			))

			Expect(snapshot.Resources).To(Equal(
				linux_backend.ResourcesSnapshot{
					UID:     containerResources.UID,
					Network: containerResources.Network,
					Ports:   containerResources.Ports,
				},
			))

			Expect(snapshot.NetIns).To(Equal(
				[]linux_backend.NetInSpec{
					{
						HostPort:      1,
						ContainerPort: 2,
					},
					{
						HostPort:      3,
						ContainerPort: 4,
					},
				},
			))

			Expect(snapshot.NetOuts).To(Equal(
				[]linux_backend.NetOutSpec{
					{
						Network: "network-a",
						Port:    1,
					},
					{
						Network: "network-b",
						Port:    2,
					},
				},
			))

			Expect(snapshot.Jobs).To(ContainElement(
				linux_backend.JobSnapshot{
					ID:            0,
					DiscardOutput: true,
				},
			))

			Expect(snapshot.Jobs).To(ContainElement(
				linux_backend.JobSnapshot{
					ID:            1,
					DiscardOutput: false,
				},
			))
		})

		Context("with no limits set", func() {
			It("saves them as nil, not zero values", func() {
				var err error

				out := new(bytes.Buffer)

				err = container.Snapshot(out)
				Expect(err).ToNot(HaveOccurred())

				var snapshot linux_backend.ContainerSnapshot

				err = json.NewDecoder(out).Decode(&snapshot)
				Expect(err).ToNot(HaveOccurred())

				Expect(snapshot.Limits).To(Equal(
					linux_backend.LimitsSnapshot{
						Memory:    nil,
						Disk:      nil,
						Bandwidth: nil,
						CPU:       nil,
					},
				))
			})
		})
	})

	Describe("Restoring", func() {
		It("sets the container's state and events", func() {
			err := container.Restore(linux_backend.ContainerSnapshot{
				State:  "active",
				Events: []string{"out of memory", "foo"},
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(container.State()).To(Equal(linux_backend.State("active")))
			Expect(container.Events()).To(Equal([]string{
				"out of memory",
				"foo",
			}))
		})

		It("restores job state", func() {
			fakeRunner.WhenRunning(
				fake_command_runner.CommandSpec{
					Path: "/depot/some-id/bin/iomux-link",
					Args: []string{
						"-w", "/depot/some-id/jobs/0/cursors",
						"/depot/some-id/jobs/0",
					},
				},
				func(cmd *exec.Cmd) error {
					cmd.Stdout.Write([]byte("hello\n"))
					time.Sleep(500 * time.Millisecond)
					return nil
				},
			)

			fakeRunner.WhenRunning(
				fake_command_runner.CommandSpec{
					Path: "/depot/some-id/bin/iomux-link",
					Args: []string{
						"-w", "/depot/some-id/jobs/1/cursors",
						"/depot/some-id/jobs/1",
					},
				},
				func(cmd *exec.Cmd) error {
					cmd.Stdout.Write([]byte("goodbye\n"))
					time.Sleep(500 * time.Millisecond)
					return nil
				},
			)

			err := container.Restore(linux_backend.ContainerSnapshot{
				State:  "active",
				Events: []string{},

				Jobs: []linux_backend.JobSnapshot{
					{
						ID:            0,
						DiscardOutput: false,
					},
					{
						ID:            1,
						DiscardOutput: true,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			linked := make(chan bool)

			go func() {
				res, err := container.Link(0)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Stdout).To(Equal([]byte("hello\n")))
				linked <- true
			}()

			go func() {
				res, err := container.Link(1)
				Expect(err).ToNot(HaveOccurred())
				Expect(res.Stdout).To(BeEmpty())
				linked <- true
			}()

			<-linked
			<-linked
		})

		It("starts new job IDs after the highest restored ID", func() {
			err := container.Restore(linux_backend.ContainerSnapshot{
				State:  "active",
				Events: []string{},

				Jobs: []linux_backend.JobSnapshot{
					{
						ID:            0,
						DiscardOutput: false,
					},
					{
						ID:            1,
						DiscardOutput: true,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			setupSuccessfulSpawn()

			jobID, err := container.Spawn(backend.JobSpec{})
			Expect(err).ToNot(HaveOccurred())
			Expect(jobID).To(Equal(uint32(2)))
		})

		It("redoes network setup and net-in/net-outs", func() {
			err := container.Restore(linux_backend.ContainerSnapshot{
				State:  "active",
				Events: []string{},

				NetIns: []linux_backend.NetInSpec{
					{
						HostPort:      1234,
						ContainerPort: 5678,
					},
					{
						HostPort:      1235,
						ContainerPort: 5679,
					},
				},

				NetOuts: []linux_backend.NetOutSpec{
					{
						Network: "somehost.example.com",
						Port:    80,
					},
					{
						Network: "someotherhost.example.com",
						Port:    8080,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeRunner).To(HaveExecutedSerially(
				fake_command_runner.CommandSpec{
					Path: "/depot/some-id/net.sh",
					Args: []string{"setup"},
				},
				fake_command_runner.CommandSpec{
					Path: "/depot/some-id/net.sh",
					Args: []string{"in"},
				},
				fake_command_runner.CommandSpec{
					Path: "/depot/some-id/net.sh",
					Args: []string{"in"},
				},
				fake_command_runner.CommandSpec{
					Path: "/depot/some-id/net.sh",
					Args: []string{"out"},
				},
				fake_command_runner.CommandSpec{
					Path: "/depot/some-id/net.sh",
					Args: []string{"out"},
				},
			))
		})

		for _, cmd := range []string{"setup", "in", "out"} {
			command := cmd

			Context("when net.sh "+cmd+" fails", func() {
				disaster := errors.New("oh no!")

				BeforeEach(func() {
					fakeRunner.WhenRunning(
						fake_command_runner.CommandSpec{
							Path: "/depot/some-id/net.sh",
							Args: []string{command},
						}, func(*exec.Cmd) error {
							return disaster
						},
					)
				})

				It("returns the error", func() {
					err := container.Restore(linux_backend.ContainerSnapshot{
						State:  "active",
						Events: []string{},

						NetIns: []linux_backend.NetInSpec{
							{
								HostPort:      1234,
								ContainerPort: 5678,
							},
							{
								HostPort:      1235,
								ContainerPort: 5679,
							},
						},

						NetOuts: []linux_backend.NetOutSpec{
							{
								Network: "somehost.example.com",
								Port:    80,
							},
							{
								Network: "someotherhost.example.com",
								Port:    8080,
							},
						},
					})
					Expect(err).To(Equal(disaster))
				})
			})
		}

		It("re-enforces the memory limit", func() {
			err := container.Restore(linux_backend.ContainerSnapshot{
				State:  "active",
				Events: []string{},

				Limits: linux_backend.LimitsSnapshot{
					Memory: &backend.MemoryLimits{
						LimitInBytes: 1024,
					},
				},
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeCgroups.SetValues()).To(ContainElement(
				fake_cgroups_manager.SetValue{
					Subsystem: "memory",
					Name:      "memory.limit_in_bytes",
					Value:     "1024",
				},
			))

			Expect(fakeCgroups.SetValues()).To(ContainElement(
				fake_cgroups_manager.SetValue{
					Subsystem: "memory",
					Name:      "memory.memsw.limit_in_bytes",
					Value:     "1024",
				},
			))

			// oom will exit immediately as the command runner is faked out
			Eventually(container.Events).Should(ContainElement("out of memory"))
		})

		Context("when no memory limit is present", func() {
			It("does not set a limit", func() {
				err := container.Restore(linux_backend.ContainerSnapshot{
					State:  "active",
					Events: []string{},
				})
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeCgroups.SetValues()).To(BeEmpty())
			})
		})

		Context("when re-enforcing the memory limit fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeCgroups.WhenSetting("memory", "memory.limit_in_bytes", func() error {
					return disaster
				})
			})

			It("returns the error", func() {
				err := container.Restore(linux_backend.ContainerSnapshot{
					State:  "active",
					Events: []string{},

					Limits: linux_backend.LimitsSnapshot{
						Memory: &backend.MemoryLimits{
							LimitInBytes: 1024,
						},
					},
				})
				Expect(err).To(Equal(disaster))
			})
		})
	})

	Describe("Starting", func() {
		It("executes the container's start.sh with the correct environment", func() {
			err := container.Start()
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeRunner).To(HaveExecutedSerially(
				fake_command_runner.CommandSpec{
					Path: "/depot/some-id/start.sh",
					Env: []string{
						"id=some-id",
						"container_iface_mtu=1500",
						"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
					},
				},
			))
		})

		It("changes the container's state to active", func() {
			Expect(container.State()).To(Equal(linux_backend.StateBorn))

			err := container.Start()
			Expect(err).ToNot(HaveOccurred())

			Expect(container.State()).To(Equal(linux_backend.StateActive))
		})

		Context("when start.sh fails", func() {
			nastyError := errors.New("oh no!")

			BeforeEach(func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/start.sh",
					}, func(*exec.Cmd) error {
						return nastyError
					},
				)
			})

			It("returns the error", func() {
				err := container.Start()
				Expect(err).To(Equal(nastyError))
			})

			It("does not change the container's state", func() {
				Expect(container.State()).To(Equal(linux_backend.StateBorn))

				err := container.Start()
				Expect(err).To(HaveOccurred())

				Expect(container.State()).To(Equal(linux_backend.StateBorn))
			})
		})
	})

	Describe("Stopping", func() {
		It("executes the container's stop.sh", func() {
			err := container.Stop(false)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeRunner).To(HaveExecutedSerially(
				fake_command_runner.CommandSpec{
					Path: "/depot/some-id/stop.sh",
				},
			))
		})

		It("sets the container's state to stopped", func() {
			Expect(container.State()).To(Equal(linux_backend.StateBorn))

			err := container.Stop(false)
			Expect(err).ToNot(HaveOccurred())

			Expect(container.State()).To(Equal(linux_backend.StateStopped))

		})

		Context("when kill is true", func() {
			It("executes stop.sh with -w 0", func() {
				err := container.Stop(true)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeRunner).To(HaveExecutedSerially(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/stop.sh",
						Args: []string{"-w", "0"},
					},
				))
			})
		})

		Context("when stop.sh fails", func() {
			nastyError := errors.New("oh no!")

			BeforeEach(func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/stop.sh",
					}, func(*exec.Cmd) error {
						return nastyError
					},
				)
			})

			It("returns the error", func() {
				err := container.Stop(false)
				Expect(err).To(Equal(nastyError))
			})

			It("does not change the container's state", func() {
				Expect(container.State()).To(Equal(linux_backend.StateBorn))

				err := container.Stop(false)
				Expect(err).To(HaveOccurred())

				Expect(container.State()).To(Equal(linux_backend.StateBorn))
			})
		})

		Context("when the container has an oom notifier running", func() {
			BeforeEach(func() {
				err := container.LimitMemory(backend.MemoryLimits{
					LimitInBytes: 42,
				})

				Expect(err).ToNot(HaveOccurred())
			})

			It("stops it", func() {
				err := container.Stop(false)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeRunner).To(HaveKilled(fake_command_runner.CommandSpec{
					Path: "/depot/some-id/bin/oom",
				}))
			})
		})
	})

	Describe("Cleaning up", func() {
		Context("when the container has an oom notifier running", func() {
			BeforeEach(func() {
				err := container.LimitMemory(backend.MemoryLimits{
					LimitInBytes: 42,
				})

				Expect(err).ToNot(HaveOccurred())
			})

			It("stops it", func() {
				container.Cleanup()

				Expect(fakeRunner).To(HaveKilled(fake_command_runner.CommandSpec{
					Path: "/depot/some-id/bin/oom",
				}))
			})
		})

		Context("when there are active jobs", func() {
			linked := make(chan bool)

			BeforeEach(func() {
				setupSuccessfulSpawn()

				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/bin/iomux-link",
					}, func(*exec.Cmd) error {
						linked <- true
						select {}
						return nil
					},
				)

				_, err := container.Spawn(backend.JobSpec{AutoLink: true})
				Expect(err).ToNot(HaveOccurred())

				_, err = container.Spawn(backend.JobSpec{AutoLink: true})
				Expect(err).ToNot(HaveOccurred())
			})

			It("interrupts their iomux-link", func() {
				<-linked
				<-linked

				container.Cleanup()

				Expect(fakeRunner).To(HaveSignalled(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/bin/iomux-link",
						Args: []string{
							"-w", "/depot/some-id/jobs/0/cursors",
							"/depot/some-id/jobs/0",
						},
					},
					os.Interrupt,
				))

				Expect(fakeRunner).To(HaveSignalled(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/bin/iomux-link",
						Args: []string{
							"-w", "/depot/some-id/jobs/1/cursors",
							"/depot/some-id/jobs/1",
						},
					},
					os.Interrupt,
				))
			})
		})
	})

	Describe("Copying in", func() {
		It("executes rsync from src into dst via wsh --rsh", func() {
			fakeRunner.ServerRootPath = "/host"

			err := container.CopyIn("/src", "/dst")
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeRunner).To(HaveExecutedSerially(
				fake_command_runner.CommandSpec{
					Path: "rsync",
					Args: []string{
						"-e",
						"/depot/some-id/bin/wsh --socket /depot/some-id/run/wshd.sock --rsh",
						"-r",
						"-p",
						"--links",
						"/host/src",
						"vcap@container:/dst",
					},
				},
			))
		})

		Context("when rsync fails", func() {
			nastyError := errors.New("oh no!")

			BeforeEach(func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "rsync",
					}, func(*exec.Cmd) error {
						return nastyError
					},
				)
			})

			It("returns the error", func() {
				err := container.CopyIn("/src", "/dst")
				Expect(err).To(Equal(nastyError))
			})
		})
	})

	Describe("Copying out", func() {
		It("rsyncs from vcap@container:/src to /dst", func() {
			fakeRunner.ServerRootPath = "/host"

			err := container.CopyOut("/src", "/dst", "")
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeRunner).To(HaveExecutedSerially(
				fake_command_runner.CommandSpec{
					Path: "rsync",
					Args: []string{
						"-e",
						"/depot/some-id/bin/wsh --socket /depot/some-id/run/wshd.sock --rsh",
						"-r",
						"-p",
						"--links",
						"vcap@container:/src",
						"/host/dst",
					},
				},
			))
		})

		Context("when an owner is given", func() {
			It("chowns the files after rsyncing", func() {
				fakeRunner.ServerRootPath = "/host"

				err := container.CopyOut("/src", "/dst", "some-user")
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeRunner).To(HaveExecutedSerially(
					fake_command_runner.CommandSpec{
						Path: "rsync",
					},
					fake_command_runner.CommandSpec{
						Path: "chown",
						Args: []string{"-R", "some-user", "/host/dst"},
					},
				))
			})
		})

		Context("when rsync fails", func() {
			nastyError := errors.New("oh no!")

			BeforeEach(func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "rsync",
					}, func(*exec.Cmd) error {
						return nastyError
					},
				)
			})

			It("returns the error", func() {
				err := container.CopyOut("/src", "/dst", "")
				Expect(err).To(Equal(nastyError))
			})
		})

		Context("when chowning fails", func() {
			nastyError := errors.New("oh no!")

			BeforeEach(func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "chown",
					}, func(*exec.Cmd) error {
						return nastyError
					},
				)
			})

			It("returns the error", func() {
				err := container.CopyOut("/src", "/dst", "some-user")
				Expect(err).To(Equal(nastyError))
			})
		})
	})

	Describe("Spawning", func() {
		It("runs the /bin/bash via wsh with the given script as the input, and rlimits in env", func() {
			setupSuccessfulSpawn()

			jobID, err := container.Spawn(backend.JobSpec{
				Script: "/some/script",
				Limits: backend.ResourceLimits{
					As:         uint64ptr(1),
					Core:       uint64ptr(2),
					Cpu:        uint64ptr(3),
					Data:       uint64ptr(4),
					Fsize:      uint64ptr(5),
					Locks:      uint64ptr(6),
					Memlock:    uint64ptr(7),
					Msgqueue:   uint64ptr(8),
					Nice:       uint64ptr(9),
					Nofile:     uint64ptr(10),
					Nproc:      uint64ptr(11),
					Rss:        uint64ptr(12),
					Rtprio:     uint64ptr(13),
					Sigpending: uint64ptr(14),
					Stack:      uint64ptr(15),
				},
			})

			Expect(err).ToNot(HaveOccurred())

			Eventually(fakeRunner).Should(HaveStartedExecuting(
				fake_command_runner.CommandSpec{
					Path: "/depot/some-id/bin/iomux-spawn",
					Args: []string{
						fmt.Sprintf("/depot/some-id/jobs/%d", jobID),
						"/depot/some-id/bin/wsh",
						"--socket", "/depot/some-id/run/wshd.sock",
						"--user", "vcap",
						"/bin/bash",
					},
					Stdin: "/some/script",
					Env: []string{
						"RLIMIT_AS=1",
						"RLIMIT_CORE=2",
						"RLIMIT_CPU=3",
						"RLIMIT_DATA=4",
						"RLIMIT_FSIZE=5",
						"RLIMIT_LOCKS=6",
						"RLIMIT_MEMLOCK=7",
						"RLIMIT_MSGQUEUE=8",
						"RLIMIT_NICE=9",
						"RLIMIT_NOFILE=10",
						"RLIMIT_NPROC=11",
						"RLIMIT_RSS=12",
						"RLIMIT_RTPRIO=13",
						"RLIMIT_SIGPENDING=14",
						"RLIMIT_STACK=15",
					},
				},
			))
		})

		Context("when not all rlimits are set", func() {
			It("only sets the given rlimits", func() {
				setupSuccessfulSpawn()

				jobID, err := container.Spawn(backend.JobSpec{
					Script: "/some/script",
					Limits: backend.ResourceLimits{
						As:      uint64ptr(1),
						Cpu:     uint64ptr(3),
						Fsize:   uint64ptr(5),
						Memlock: uint64ptr(7),
						Nice:    uint64ptr(9),
						Nproc:   uint64ptr(11),
						Rtprio:  uint64ptr(13),
						Stack:   uint64ptr(15),
					},
				})

				Expect(err).ToNot(HaveOccurred())

				Eventually(fakeRunner).Should(HaveStartedExecuting(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/bin/iomux-spawn",
						Args: []string{
							fmt.Sprintf("/depot/some-id/jobs/%d", jobID),
							"/depot/some-id/bin/wsh",
							"--socket", "/depot/some-id/run/wshd.sock",
							"--user", "vcap",
							"/bin/bash",
						},
						Stdin: "/some/script",
						Env: []string{
							"RLIMIT_AS=1",
							"RLIMIT_CPU=3",
							"RLIMIT_FSIZE=5",
							"RLIMIT_MEMLOCK=7",
							"RLIMIT_NICE=9",
							"RLIMIT_NPROC=11",
							"RLIMIT_RTPRIO=13",
							"RLIMIT_STACK=15",
						},
					},
				))
			})
		})

		It("returns a unique job ID", func() {
			setupSuccessfulSpawn()

			jobID1, err := container.Spawn(backend.JobSpec{
				Script: "/some/script",
			})
			Expect(err).ToNot(HaveOccurred())

			jobID2, err := container.Spawn(backend.JobSpec{
				Script: "/some/script",
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(jobID1).ToNot(Equal(jobID2))
		})

		Context("with 'privileged' true", func() {
			BeforeEach(setupSuccessfulSpawn)

			It("runs with --user root", func() {
				jobID, err := container.Spawn(backend.JobSpec{
					Script:     "/some/script",
					Privileged: true,
				})

				Expect(err).ToNot(HaveOccurred())

				Eventually(fakeRunner).Should(HaveStartedExecuting(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/bin/iomux-spawn",
						Args: []string{
							fmt.Sprintf("/depot/some-id/jobs/%d", jobID),
							"/depot/some-id/bin/wsh",
							"--socket", "/depot/some-id/run/wshd.sock",
							"--user", "root",
							"/bin/bash",
						},
						Stdin: "/some/script",
					},
				))
			})
		})

		Context("when spawning fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/bin/iomux-spawn",
					}, func(*exec.Cmd) error {
						return disaster
					},
				)
			})

			It("returns the error", func() {
				_, err := container.Spawn(backend.JobSpec{
					Script:     "/some/script",
					Privileged: true,
				})

				Expect(err).To(Equal(disaster))
			})
		})
	})

	Describe("Linking", func() {
		BeforeEach(setupSuccessfulSpawn)

		Context("to a started job", func() {
			BeforeEach(func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/bin/iomux-link",
					}, func(cmd *exec.Cmd) error {
						cmd.Stdout.Write([]byte("hi out\n"))
						cmd.Stderr.Write([]byte("hi err\n"))

						dummyCmd := exec.Command("/bin/bash", "-c", "exit 42")
						dummyCmd.Run()

						cmd.ProcessState = dummyCmd.ProcessState

						return nil
					},
				)
			})

			It("returns the exit status, stdout, and stderr", func() {
				jobID, err := container.Spawn(backend.JobSpec{
					Script: "/some/script",
				})
				Expect(err).ToNot(HaveOccurred())

				jobResult, err := container.Link(jobID)
				Expect(err).ToNot(HaveOccurred())
				Expect(jobResult.ExitStatus).To(Equal(uint32(42)))
				Expect(jobResult.Stdout).To(Equal([]byte("hi out\n")))
				Expect(jobResult.Stderr).To(Equal([]byte("hi err\n")))
			})

			Context("with output discarded", func() {
				It("returns the exit status but not stdout or stderr", func() {
					jobID, err := container.Spawn(backend.JobSpec{
						Script:        "/some/script",
						DiscardOutput: true,
					})
					Expect(err).ToNot(HaveOccurred())

					jobResult, err := container.Link(jobID)
					Expect(err).ToNot(HaveOccurred())
					Expect(jobResult.ExitStatus).To(Equal(uint32(42)))
					Expect(jobResult.Stdout).To(BeEmpty())
					Expect(jobResult.Stderr).To(BeEmpty())
				})
			})
		})

		Context("to a job that has already completed", func() {
			It("returns an error", func() {
				jobID, err := container.Spawn(backend.JobSpec{
					Script: "/some/script",
				})
				Expect(err).ToNot(HaveOccurred())

				_, err = container.Link(jobID)
				Expect(err).ToNot(HaveOccurred())

				_, err = container.Link(jobID)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("to an unknown job", func() {
			It("returns an error", func() {
				_, err := container.Link(42)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Streaming", func() {
		BeforeEach(setupSuccessfulSpawn)

		Context("a started job", func() {
			BeforeEach(func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/bin/iomux-link",
					}, func(cmd *exec.Cmd) error {
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

			It("streams stderr and stdout and exit status", func(done Done) {
				jobID, err := container.Spawn(backend.JobSpec{
					Script:   "/some/script",
					AutoLink: true,
				})
				Expect(err).ToNot(HaveOccurred())

				jobStreamChannel, err := container.Stream(jobID)
				Expect(err).ToNot(HaveOccurred())

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

				chunk3 := <-jobStreamChannel
				Expect(chunk3.Name).To(Equal(""))
				Expect(string(chunk3.Data)).To(Equal(""))
				Expect(chunk3.ExitStatus).ToNot(BeNil())
				Expect(*chunk3.ExitStatus).To(Equal(uint32(42)))
				//Expect(chunk3.Info).ToNot(BeNil())

				close(done)
			}, 5.0)
		})

		Context("a job that has already completed", func() {
			It("returns an error", func() {
				jobID, err := container.Spawn(backend.JobSpec{
					Script: "/some/script",
				})
				Expect(err).ToNot(HaveOccurred())

				_, err = container.Link(jobID)
				Expect(err).ToNot(HaveOccurred())

				_, err = container.Stream(jobID)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("an unknown job", func() {
			It("returns an error", func() {
				_, err := container.Stream(42)
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("Limiting bandwidth", func() {
		limits := backend.BandwidthLimits{
			RateInBytesPerSecond:      128,
			BurstRateInBytesPerSecond: 256,
		}

		It("sets the limit via the bandwidth manager with the new limits", func() {
			err := container.LimitBandwidth(limits)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeBandwidthManager.EnforcedLimits).To(ContainElement(limits))
		})

		Context("when setting the limit fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeBandwidthManager.SetLimitsError = disaster
			})

			It("returns the error", func() {
				err := container.LimitBandwidth(limits)
				Expect(err).To(Equal(disaster))
			})
		})
	})

	Describe("Getting the current bandwidth limit", func() {
		limits := backend.BandwidthLimits{
			RateInBytesPerSecond:      128,
			BurstRateInBytesPerSecond: 256,
		}

		It("returns a zero value if no limits are set", func() {
			receivedLimits, err := container.CurrentBandwidthLimits()
			Expect(err).ToNot(HaveOccurred())
			Expect(receivedLimits).To(BeZero())
		})

		Context("when limits are set", func() {
			It("returns them", func() {
				err := container.LimitBandwidth(limits)
				Expect(err).ToNot(HaveOccurred())

				receivedLimits, err := container.CurrentBandwidthLimits()
				Expect(err).ToNot(HaveOccurred())
				Expect(receivedLimits).To(Equal(limits))
			})

			Context("when limits fail to be set", func() {
				disaster := errors.New("oh no!")

				BeforeEach(func() {
					fakeBandwidthManager.SetLimitsError = disaster
				})

				It("does not update the current limits", func() {
					err := container.LimitBandwidth(limits)
					Expect(err).To(Equal(disaster))

					receivedLimits, err := container.CurrentBandwidthLimits()
					Expect(err).ToNot(HaveOccurred())
					Expect(receivedLimits).To(BeZero())
				})
			})
		})
	})

	Describe("Limiting memory", func() {
		It("starts the oom notifier", func() {
			limits := backend.MemoryLimits{
				LimitInBytes: 102400,
			}

			err := container.LimitMemory(limits)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeRunner).To(HaveStartedExecuting(
				fake_command_runner.CommandSpec{
					Path: "/depot/some-id/bin/oom",
					Args: []string{"/cgroups/memory/instance-some-id"},
				},
			))
		})

		It("sets memory.limit_in_bytes and then memory.memsw.limit_in_bytes", func() {
			limits := backend.MemoryLimits{
				LimitInBytes: 102400,
			}

			err := container.LimitMemory(limits)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeCgroups.SetValues()).To(Equal(
				[]fake_cgroups_manager.SetValue{
					{
						Subsystem: "memory",
						Name:      "memory.limit_in_bytes",
						Value:     "102400",
					},
					{
						Subsystem: "memory",
						Name:      "memory.memsw.limit_in_bytes",
						Value:     "102400",
					},
					{
						Subsystem: "memory",
						Name:      "memory.limit_in_bytes",
						Value:     "102400",
					},
				},
			))
		})

		Context("when the oom notifier is already running", func() {
			It("does not start another", func() {
				started := 0

				fakeRunner.WhenRunning(fake_command_runner.CommandSpec{
					Path: "/depot/some-id/bin/oom",
				}, func(*exec.Cmd) error {
					started++
					return nil
				})

				limits := backend.MemoryLimits{
					LimitInBytes: 102400,
				}

				err := container.LimitMemory(limits)
				Expect(err).ToNot(HaveOccurred())

				err = container.LimitMemory(limits)
				Expect(err).ToNot(HaveOccurred())

				Expect(started).To(Equal(1))
			})
		})

		Context("when the oom notifier exits 0", func() {
			BeforeEach(func() {
				fakeRunner.WhenWaitingFor(fake_command_runner.CommandSpec{
					Path: "/depot/some-id/bin/oom",
				}, func(cmd *exec.Cmd) error {
					return nil
				})
			})

			It("stops the container", func() {
				limits := backend.MemoryLimits{
					LimitInBytes: 102400,
				}

				err := container.LimitMemory(limits)
				Expect(err).ToNot(HaveOccurred())

				Eventually(fakeRunner).Should(HaveExecutedSerially(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/stop.sh",
					},
				))
			})

			It("registers an 'out of memory' event", func() {
				limits := backend.MemoryLimits{
					LimitInBytes: 102400,
				}

				err := container.LimitMemory(limits)
				Expect(err).ToNot(HaveOccurred())

				Eventually(func() []string {
					return container.Events()
				}).Should(ContainElement("out of memory"))
			})
		})

		Context("when setting memory.memsw.limit_in_bytes fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeCgroups.WhenSetting("memory", "memory.memsw.limit_in_bytes", func() error {
					return disaster
				})
			})

			It("does not fail", func() {
				err := container.LimitMemory(backend.MemoryLimits{
					LimitInBytes: 102400,
				})

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when setting memory.limit_in_bytes fails only the first time", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				numSet := 0

				fakeCgroups.WhenSetting("memory", "memory.limit_in_bytes", func() error {
					numSet++

					if numSet == 1 {
						return disaster
					}

					return nil
				})
			})

			It("succeeds", func() {
				fakeCgroups.WhenGetting("memory", "memory.limit_in_bytes", func() (string, error) {
					return "123", nil
				})

				err := container.LimitMemory(backend.MemoryLimits{
					LimitInBytes: 102400,
				})

				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when setting memory.limit_in_bytes fails the second time", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				numSet := 0

				fakeCgroups.WhenSetting("memory", "memory.limit_in_bytes", func() error {
					numSet++

					if numSet == 2 {
						return disaster
					}

					return nil
				})
			})

			It("returns the error and no limits", func() {
				err := container.LimitMemory(backend.MemoryLimits{
					LimitInBytes: 102400,
				})

				Expect(err).To(Equal(disaster))
			})
		})

		Context("when starting the oom notifier fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeRunner.WhenRunning(fake_command_runner.CommandSpec{
					Path: "/depot/some-id/bin/oom",
				}, func(cmd *exec.Cmd) error {
					return disaster
				})
			})

			It("returns the error", func() {
				err := container.LimitMemory(backend.MemoryLimits{
					LimitInBytes: 102400,
				})

				Expect(err).To(Equal(disaster))
			})
		})
	})

	Describe("Getting the current memory limit", func() {
		It("returns the limited memory", func() {
			fakeCgroups.WhenGetting("memory", "memory.limit_in_bytes", func() (string, error) {
				return "18446744073709551615", nil
			})

			limits, err := container.CurrentMemoryLimits()
			Expect(err).ToNot(HaveOccurred())
			Expect(limits.LimitInBytes).To(Equal(uint64(math.MaxUint64)))
		})

		Context("when getting the limit fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeCgroups.WhenGetting("memory", "memory.limit_in_bytes", func() (string, error) {
					return "", disaster
				})
			})

			It("returns the error", func() {
				limits, err := container.CurrentMemoryLimits()
				Expect(err).To(Equal(disaster))
				Expect(limits).To(BeZero())
			})
		})
	})

	Describe("Limiting CPU", func() {
		It("sets cpu.shares", func() {
			limits := backend.CPULimits{
				LimitInShares: 512,
			}

			err := container.LimitCPU(limits)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeCgroups.SetValues()).To(Equal(
				[]fake_cgroups_manager.SetValue{
					{
						Subsystem: "cpu",
						Name:      "cpu.shares",
						Value:     "512",
					},
				},
			))
		})

		Context("when setting cpu.shares fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeCgroups.WhenSetting("cpu", "cpu.shares", func() error {
					return disaster
				})
			})

			It("returns the error", func() {
				err := container.LimitCPU(backend.CPULimits{
					LimitInShares: 512,
				})

				Expect(err).To(Equal(disaster))
			})
		})
	})

	Describe("Getting the current CPU limits", func() {
		It("returns the CPU limits", func() {
			fakeCgroups.WhenGetting("cpu", "cpu.shares", func() (string, error) {
				return "512", nil
			})

			limits, err := container.CurrentCPULimits()
			Expect(err).ToNot(HaveOccurred())
			Expect(limits.LimitInShares).To(Equal(uint64(512)))
		})

		Context("when getting the limit fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeCgroups.WhenGetting("cpu", "cpu.shares", func() (string, error) {
					return "", disaster
				})
			})

			It("returns the error", func() {
				limits, err := container.CurrentCPULimits()
				Expect(err).To(Equal(disaster))
				Expect(limits).To(BeZero())
			})
		})
	})

	Describe("Limiting disk", func() {
		limits := backend.DiskLimits{
			BlockLimit: 1,
			Block:      2,
			BlockSoft:  3,
			BlockHard:  4,

			InodeLimit: 11,
			Inode:      12,
			InodeSoft:  13,
			InodeHard:  14,

			ByteLimit: 21,
			Byte:      22,
			ByteSoft:  23,
			ByteHard:  24,
		}

		It("sets the quota via the quota manager with the uid and limits", func() {
			resultingLimits := backend.DiskLimits{
				Block: 1234567,
			}

			fakeQuotaManager.GetLimitsResult = resultingLimits

			err := container.LimitDisk(limits)
			Expect(err).ToNot(HaveOccurred())

			uid := containerResources.UID

			Expect(fakeQuotaManager.Limited).To(HaveKey(uid))
			Expect(fakeQuotaManager.Limited[uid]).To(Equal(limits))
		})

		Context("when setting the quota fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeQuotaManager.SetLimitsError = disaster
			})

			It("returns the error", func() {
				err := container.LimitDisk(limits)
				Expect(err).To(Equal(disaster))
			})
		})
	})

	Describe("Getting the current disk limits", func() {
		It("returns the disk limits", func() {
			limits := backend.DiskLimits{
				Block: 1234567,
			}

			fakeQuotaManager.GetLimitsResult = limits

			receivedLimits, err := container.CurrentDiskLimits()
			Expect(err).ToNot(HaveOccurred())
			Expect(receivedLimits).To(Equal(limits))
		})

		Context("when getting the limit fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeQuotaManager.GetLimitsError = disaster
			})

			It("returns the error", func() {
				limits, err := container.CurrentDiskLimits()
				Expect(err).To(Equal(disaster))
				Expect(limits).To(BeZero())
			})
		})
	})

	Describe("Net in", func() {
		It("executes net.sh in with HOST_PORT and CONTAINER_PORT", func() {
			hostPort, containerPort, err := container.NetIn(123, 456)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeRunner).To(HaveExecutedSerially(
				fake_command_runner.CommandSpec{
					Path: "/depot/some-id/net.sh",
					Args: []string{"in"},
					Env: []string{
						"HOST_PORT=123",
						"CONTAINER_PORT=456",
					},
				},
			))

			Expect(hostPort).To(Equal(uint32(123)))
			Expect(containerPort).To(Equal(uint32(456)))
		})

		Context("when a host port is not provided", func() {
			It("acquires one from the port pool", func() {
				hostPort, containerPort, err := container.NetIn(0, 456)
				Expect(err).ToNot(HaveOccurred())

				Expect(hostPort).To(Equal(uint32(1000)))
				Expect(containerPort).To(Equal(uint32(456)))

				secondHostPort, _, err := container.NetIn(0, 456)
				Expect(err).ToNot(HaveOccurred())

				Expect(secondHostPort).ToNot(Equal(hostPort))

				Expect(container.Resources().Ports).To(ContainElement(hostPort))
			})

			Context("and acquiring a port from the pool fails", func() {
				disaster := errors.New("oh no!")

				BeforeEach(func() {
					fakePortPool.AcquireError = disaster
				})

				It("returns the error", func() {
					_, _, err := container.NetIn(0, 456)
					Expect(err).To(Equal(disaster))
				})
			})
		})

		Context("when a container port is not provided", func() {
			It("defaults it to the host port", func() {
				hostPort, containerPort, err := container.NetIn(123, 0)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeRunner).To(HaveExecutedSerially(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/net.sh",
						Args: []string{"in"},
						Env: []string{
							"HOST_PORT=123",
							"CONTAINER_PORT=123",
						},
					},
				))

				Expect(hostPort).To(Equal(uint32(123)))
				Expect(containerPort).To(Equal(uint32(123)))
			})

			Context("and a host port is not provided either", func() {
				It("defaults it to the same acquired port", func() {
					hostPort, containerPort, err := container.NetIn(0, 0)
					Expect(err).ToNot(HaveOccurred())

					Expect(fakeRunner).To(HaveExecutedSerially(
						fake_command_runner.CommandSpec{
							Path: "/depot/some-id/net.sh",
							Args: []string{"in"},
							Env: []string{
								"HOST_PORT=1000",
								"CONTAINER_PORT=1000",
							},
						},
					))

					Expect(hostPort).To(Equal(uint32(1000)))
					Expect(containerPort).To(Equal(uint32(1000)))
				})
			})
		})

		Context("when net.sh fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/net.sh",
					}, func(*exec.Cmd) error {
						return disaster
					},
				)
			})

			It("returns the error", func() {
				_, _, err := container.NetIn(123, 456)
				Expect(err).To(Equal(disaster))
			})
		})
	})

	Describe("Net out", func() {
		It("executes net.sh out with NETWORK and PORT", func() {
			err := container.NetOut("1.2.3.4/22", 567)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeRunner).To(HaveExecutedSerially(
				fake_command_runner.CommandSpec{
					Path: "/depot/some-id/net.sh",
					Args: []string{"out"},
					Env: []string{
						"NETWORK=1.2.3.4/22",
						"PORT=567",
					},
				},
			))
		})

		Context("when port 0 is given", func() {
			It("executes with PORT as an empty string", func() {
				err := container.NetOut("1.2.3.4/22", 0)
				Expect(err).ToNot(HaveOccurred())

				Expect(fakeRunner).To(HaveExecutedSerially(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/net.sh",
						Args: []string{"out"},
						Env: []string{
							"NETWORK=1.2.3.4/22",
							"PORT=",
						},
					},
				))
			})

			Context("and a network is not given", func() {
				It("returns an error", func() {
					err := container.NetOut("", 0)
					Expect(err).To(HaveOccurred())
				})
			})
		})

		Context("when net.sh fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/net.sh",
					}, func(*exec.Cmd) error {
						return disaster
					},
				)
			})

			It("returns the error", func() {
				err := container.NetOut("1.2.3.4/22", 567)
				Expect(err).To(Equal(disaster))
			})
		})
	})

	Describe("Info", func() {
		It("returns the container's state", func() {
			info, err := container.Info()
			Expect(err).ToNot(HaveOccurred())

			Expect(info.State).To(Equal("born"))
		})

		It("returns the container's events", func() {
			info, err := container.Info()
			Expect(err).ToNot(HaveOccurred())

			Expect(info.Events).To(Equal([]string{}))
		})

		It("returns the container's network info", func() {
			info, err := container.Info()
			Expect(err).ToNot(HaveOccurred())

			Expect(info.HostIP).To(Equal("10.254.0.1"))
			Expect(info.ContainerIP).To(Equal("10.254.0.2"))
		})

		It("returns the container's path", func() {
			info, err := container.Info()
			Expect(err).ToNot(HaveOccurred())
			Expect(info.ContainerPath).To(Equal("/depot/some-id"))
		})

		Context("with running jobs", func() {
			BeforeEach(setupSuccessfulSpawn)

			It("returns their job IDs", func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "/depot/some-id/bin/iomux-link",
					},
					func(cmd *exec.Cmd) error {
						// block forever so the job remains active
						select {}

						return nil
					},
				)

				jobID1, err := container.Spawn(backend.JobSpec{
					Script: "/some/script",
				})
				Expect(err).ToNot(HaveOccurred())

				jobID2, err := container.Spawn(backend.JobSpec{
					Script: "/some/script",
				})
				Expect(err).ToNot(HaveOccurred())

				info, err := container.Info()
				Expect(err).ToNot(HaveOccurred())
				Expect(info.JobIDs).To(Equal([]uint32{jobID1, jobID2}))
			})
		})

		Describe("memory info", func() {
			BeforeEach(func() {
				fakeCgroups.WhenGetting("memory", "memory.stat", func() (string, error) {
					return `cache 1
rss 2
mapped_file 3
pgpgin 4
pgpgout 5
swap 6
pgfault 7
pgmajfault 8
inactive_anon 9
active_anon 10
inactive_file 11
active_file 12
unevictable 13
hierarchical_memory_limit 14
hierarchical_memsw_limit 15
total_cache 16
total_rss 17
total_mapped_file 18
total_pgpgin 19
total_pgpgout 20
total_swap 21
total_pgfault 22
total_pgmajfault 23
total_inactive_anon 24
total_active_anon 25
total_inactive_file 26
total_active_file 27
total_unevictable 28
`, nil
				})
			})

			It("is returned in the response", func() {
				info, err := container.Info()
				Expect(err).ToNot(HaveOccurred())
				Expect(info.MemoryStat).To(Equal(backend.ContainerMemoryStat{
					Cache:                   1,
					Rss:                     2,
					MappedFile:              3,
					Pgpgin:                  4,
					Pgpgout:                 5,
					Swap:                    6,
					Pgfault:                 7,
					Pgmajfault:              8,
					InactiveAnon:            9,
					ActiveAnon:              10,
					InactiveFile:            11,
					ActiveFile:              12,
					Unevictable:             13,
					HierarchicalMemoryLimit: 14,
					HierarchicalMemswLimit:  15,
					TotalCache:              16,
					TotalRss:                17,
					TotalMappedFile:         18,
					TotalPgpgin:             19,
					TotalPgpgout:            20,
					TotalSwap:               21,
					TotalPgfault:            22,
					TotalPgmajfault:         23,
					TotalInactiveAnon:       24,
					TotalActiveAnon:         25,
					TotalInactiveFile:       26,
					TotalActiveFile:         27,
					TotalUnevictable:        28,
				}))
			})
		})

		Context("when getting memory.stat fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeCgroups.WhenGetting("memory", "memory.stat", func() (string, error) {
					return "", disaster
				})
			})

			It("returns an error", func() {
				_, err := container.Info()
				Expect(err).To(Equal(disaster))
			})
		})

		Describe("cpu info", func() {
			BeforeEach(func() {
				fakeCgroups.WhenGetting("cpuacct", "cpuacct.usage", func() (string, error) {
					return `42
`, nil
				})

				fakeCgroups.WhenGetting("cpuacct", "cpuacct.stat", func() (string, error) {
					return `user 1
system 2
`, nil
				})
			})

			It("is returned in the response", func() {
				info, err := container.Info()
				Expect(err).ToNot(HaveOccurred())
				Expect(info.CPUStat).To(Equal(backend.ContainerCPUStat{
					Usage:  42,
					User:   1,
					System: 2,
				}))
			})
		})

		Context("when getting cpuacct/cpuacct.usage fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeCgroups.WhenGetting("cpuacct", "cpuacct.usage", func() (string, error) {
					return "", disaster
				})
			})

			It("returns an error", func() {
				_, err := container.Info()
				Expect(err).To(Equal(disaster))
			})
		})

		Context("when getting cpuacct/cpuacct.stat fails", func() {
			disaster := errors.New("oh no!")

			BeforeEach(func() {
				fakeCgroups.WhenGetting("cpuacct", "cpuacct.stat", func() (string, error) {
					return "", disaster
				})
			})

			It("returns an error", func() {
				_, err := container.Info()
				Expect(err).To(Equal(disaster))
			})
		})

		Describe("disk usage info", func() {
			It("is returned in the response", func() {
				fakeQuotaManager.GetUsageResult = backend.ContainerDiskStat{
					BytesUsed:  1,
					InodesUsed: 2,
				}

				info, err := container.Info()
				Expect(err).ToNot(HaveOccurred())

				Expect(info.DiskStat).To(Equal(backend.ContainerDiskStat{
					BytesUsed:  1,
					InodesUsed: 2,
				}))
			})

			Context("when getting the disk usage fails", func() {
				disaster := errors.New("oh no!")

				BeforeEach(func() {
					fakeQuotaManager.GetUsageError = disaster
				})

				It("returns the error", func() {
					_, err := container.Info()
					Expect(err).To(Equal(disaster))
				})
			})
		})

		Describe("bandwidth info", func() {
			It("is returned in the response", func() {
				fakeBandwidthManager.GetLimitsResult = backend.ContainerBandwidthStat{
					InRate:   1,
					InBurst:  2,
					OutRate:  3,
					OutBurst: 4,
				}

				info, err := container.Info()
				Expect(err).ToNot(HaveOccurred())

				Expect(info.BandwidthStat).To(Equal(backend.ContainerBandwidthStat{
					InRate:   1,
					InBurst:  2,
					OutRate:  3,
					OutBurst: 4,
				}))
			})

			Context("when getting the bandwidth usage fails", func() {
				disaster := errors.New("oh no!")

				BeforeEach(func() {
					fakeBandwidthManager.GetLimitsError = disaster
				})

				It("returns the error", func() {
					_, err := container.Info()
					Expect(err).To(Equal(disaster))
				})
			})
		})
	})
})

func uint64ptr(n uint64) *uint64 {
	return &n
}
