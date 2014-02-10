package container_pool_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net"
	"os/exec"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf-experimental/garden/backend"
	"github.com/pivotal-cf-experimental/garden/command_runner/fake_command_runner"
	. "github.com/pivotal-cf-experimental/garden/command_runner/fake_command_runner/matchers"
	"github.com/pivotal-cf-experimental/garden/linux_backend"
	"github.com/pivotal-cf-experimental/garden/linux_backend/container_pool"
	"github.com/pivotal-cf-experimental/garden/linux_backend/network"
	"github.com/pivotal-cf-experimental/garden/linux_backend/network_pool/fake_network_pool"
	"github.com/pivotal-cf-experimental/garden/linux_backend/port_pool/fake_port_pool"
	"github.com/pivotal-cf-experimental/garden/linux_backend/quota_manager/fake_quota_manager"
	"github.com/pivotal-cf-experimental/garden/linux_backend/uid_pool/fake_uid_pool"
)

var _ = Describe("Container pool", func() {
	var fakeRunner *fake_command_runner.FakeCommandRunner
	var fakeUIDPool *fake_uid_pool.FakeUIDPool
	var fakeNetworkPool *fake_network_pool.FakeNetworkPool
	var fakeQuotaManager *fake_quota_manager.FakeQuotaManager
	var fakePortPool *fake_port_pool.FakePortPool
	var pool *container_pool.LinuxContainerPool

	BeforeEach(func() {
		_, ipNet, err := net.ParseCIDR("1.2.0.0/20")
		Expect(err).ToNot(HaveOccurred())

		fakeUIDPool = fake_uid_pool.New(10000)

		fakeNetworkPool = fake_network_pool.New(ipNet)
		fakeRunner = fake_command_runner.New()
		fakeQuotaManager = fake_quota_manager.New()
		fakePortPool = fake_port_pool.New(1000)

		pool = container_pool.New(
			"/root/path",
			"/depot/path",
			"/rootfs/path",
			fakeUIDPool,
			fakeNetworkPool,
			fakePortPool,
			fakeRunner,
			fakeQuotaManager,
		)
	})

	Describe("setup", func() {
		It("executes setup.sh with the correct environment", func() {
			fakeQuotaManager.MountPointResult = "/depot/mount/point"

			err := pool.Setup()
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeRunner).To(HaveExecutedSerially(
				fake_command_runner.CommandSpec{
					Path: "/root/path/setup.sh",
					Env: []string{
						"POOL_NETWORK=1.2.0.0/20",
						"ALLOW_NETWORKS=",
						"DENY_NETWORKS=",
						"CONTAINER_ROOTFS_PATH=/rootfs/path",
						"CONTAINER_DEPOT_PATH=/depot/path",
						"CONTAINER_DEPOT_MOUNT_POINT_PATH=/depot/mount/point",
						"DISK_QUOTA_ENABLED=true",

						"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
					},
				},
			))
		})

		Context("when setup.sh fails", func() {
			nastyError := errors.New("oh no!")

			BeforeEach(func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "/root/path/setup.sh",
					}, func(*exec.Cmd) error {
						return nastyError
					},
				)
			})

			It("returns the error", func() {
				err := pool.Setup()
				Expect(err).To(Equal(nastyError))
			})
		})
	})

	Describe("creating", func() {
		It("returns containers with unique IDs", func() {
			container1, err := pool.Create(backend.ContainerSpec{})
			Expect(err).ToNot(HaveOccurred())

			container2, err := pool.Create(backend.ContainerSpec{})
			Expect(err).ToNot(HaveOccurred())

			Expect(container1.ID()).ToNot(Equal(container2.ID()))
		})

		It("creates containers with the correct grace time", func() {
			container, err := pool.Create(backend.ContainerSpec{
				GraceTime: 1 * time.Second,
			})
			Expect(err).ToNot(HaveOccurred())

			Expect(container.GraceTime()).To(Equal(1 * time.Second))
		})

		It("executes create.sh with the correct args and environment", func() {
			container, err := pool.Create(backend.ContainerSpec{})
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeRunner).To(HaveExecutedSerially(
				fake_command_runner.CommandSpec{
					Path: "/root/path/create.sh",
					Args: []string{"/depot/path/" + container.ID()},
					Env: []string{
						"id=" + container.ID(),
						"rootfs_path=/rootfs/path",
						"user_uid=10000",
						"network_host_ip=1.2.0.1",
						"network_container_ip=1.2.0.2",
						"network_netmask=255.255.255.252",

						"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
					},
				},
			))
		})

		Context("when bind mounts are specified", func() {
			It("appends mount commands to hook-child-before-pivot.sh", func() {
				fakeRunner.ServerRootPath = "/host"

				container, err := pool.Create(backend.ContainerSpec{
					BindMounts: []backend.BindMount{
						{
							SrcPath: "/src/path-ro",
							DstPath: "/dst/path-ro",
							Mode:    backend.BindMountModeRO,
						},
						{
							SrcPath: "/src/path-rw",
							DstPath: "/dst/path-rw",
							Mode:    backend.BindMountModeRW,
						},
					},
				})

				Expect(err).ToNot(HaveOccurred())

				containerPath := "/depot/path/" + container.ID()

				Expect(fakeRunner).To(HaveExecutedSerially(
					fake_command_runner.CommandSpec{
						Path: "bash",
						Args: []string{
							"-c",
							"echo >> " + containerPath + "/lib/hook-child-before-pivot.sh",
						},
					},
					fake_command_runner.CommandSpec{
						Path: "bash",
						Args: []string{
							"-c",
							"echo mkdir -p " + containerPath + "/mnt/dst/path-ro" +
								" >> " + containerPath + "/lib/hook-child-before-pivot.sh",
						},
					},
					fake_command_runner.CommandSpec{
						Path: "bash",
						Args: []string{
							"-c",
							"echo mount -n --bind /host/src/path-ro " + containerPath + "/mnt/dst/path-ro" +
								" >> " + containerPath + "/lib/hook-child-before-pivot.sh",
						},
					},
					fake_command_runner.CommandSpec{
						Path: "bash",
						Args: []string{
							"-c",
							"echo mount -n --bind -o remount,ro /host/src/path-ro " + containerPath + "/mnt/dst/path-ro" +
								" >> " + containerPath + "/lib/hook-child-before-pivot.sh",
						},
					},
					fake_command_runner.CommandSpec{
						Path: "bash",
						Args: []string{
							"-c",
							"echo >> " + containerPath + "/lib/hook-child-before-pivot.sh",
						},
					},
					fake_command_runner.CommandSpec{
						Path: "bash",
						Args: []string{
							"-c",
							"echo mkdir -p " + containerPath + "/mnt/dst/path-rw" +
								" >> " + containerPath + "/lib/hook-child-before-pivot.sh",
						},
					},
					fake_command_runner.CommandSpec{
						Path: "bash",
						Args: []string{
							"-c",
							"echo mount -n --bind /host/src/path-rw " + containerPath + "/mnt/dst/path-rw" +
								" >> " + containerPath + "/lib/hook-child-before-pivot.sh",
						},
					},
					fake_command_runner.CommandSpec{
						Path: "bash",
						Args: []string{
							"-c",
							"echo mount -n --bind -o remount,rw /host/src/path-rw " + containerPath + "/mnt/dst/path-rw" +
								" >> " + containerPath + "/lib/hook-child-before-pivot.sh",
						},
					},
				))
			})

			Context("when appending to hook-child-before-pivot.sh fails", func() {
				disaster := errors.New("oh no!")

				BeforeEach(func() {
					fakeRunner.WhenRunning(fake_command_runner.CommandSpec{
						Path: "bash",
					}, func(*exec.Cmd) error {
						return disaster
					})
				})

				It("returns the error", func() {
					_, err := pool.Create(backend.ContainerSpec{
						BindMounts: []backend.BindMount{
							{
								SrcPath: "/src/path-ro",
								DstPath: "/dst/path-ro",
								Mode:    backend.BindMountModeRO,
							},
							{
								SrcPath: "/src/path-rw",
								DstPath: "/dst/path-rw",
								Mode:    backend.BindMountModeRW,
							},
						},
					})

					Expect(err).To(Equal(disaster))
				})
			})
		})

		Context("when acquiring a UID fails", func() {
			nastyError := errors.New("oh no!")

			JustBeforeEach(func() {
				fakeUIDPool.AcquireError = nastyError
			})

			It("returns the error", func() {
				_, err := pool.Create(backend.ContainerSpec{})
				Expect(err).To(Equal(nastyError))
			})
		})

		Context("when acquiring a network fails", func() {
			nastyError := errors.New("oh no!")

			JustBeforeEach(func() {
				fakeNetworkPool.AcquireError = nastyError
			})

			It("returns the error and releases the uid", func() {
				_, err := pool.Create(backend.ContainerSpec{})
				Expect(err).To(Equal(nastyError))

				Expect(fakeUIDPool.Released).To(ContainElement(uint32(10000)))
			})
		})

		Context("when executing create.sh fails", func() {
			nastyError := errors.New("oh no!")

			BeforeEach(func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "/root/path/create.sh",
					}, func(*exec.Cmd) error {
						return nastyError
					},
				)
			})

			It("returns the error and releases the uid and network", func() {
				_, err := pool.Create(backend.ContainerSpec{})
				Expect(err).To(Equal(nastyError))

				Expect(fakeUIDPool.Released).To(ContainElement(uint32(10000)))
				Expect(fakeNetworkPool.Released).To(ContainElement("1.2.0.0/30"))
			})
		})
	})

	Describe("restoring", func() {
		var snapshot io.Reader

		var restoredNetwork *network.Network

		BeforeEach(func() {
			buf := new(bytes.Buffer)

			snapshot = buf

			_, ipNet, err := net.ParseCIDR("10.244.0.0/30")
			Expect(err).ToNot(HaveOccurred())

			restoredNetwork = network.New(ipNet)

			err = json.NewEncoder(buf).Encode(
				linux_backend.ContainerSnapshot{
					ID:     "some-restored-id",
					Handle: "some-restored-handle",

					GraceTime: 1 * time.Second,

					State: "some-restored-state",
					Events: []string{
						"some-restored-event",
						"some-other-restored-event",
					},

					Resources: linux_backend.ResourcesSnapshot{
						UID:     10000,
						Network: restoredNetwork,
						Ports:   []uint32{61001, 61002, 61003},
					},
				},
			)
			Expect(err).ToNot(HaveOccurred())
		})

		It("constructs a container from the snapshot", func() {
			container, err := pool.Restore(snapshot)
			Expect(err).ToNot(HaveOccurred())

			Expect(container.ID()).To(Equal("some-restored-id"))
			Expect(container.Handle()).To(Equal("some-restored-handle"))
			Expect(container.GraceTime()).To(Equal(1 * time.Second))

			linuxContainer := container.(*linux_backend.LinuxContainer)

			Expect(linuxContainer.State()).To(Equal(linux_backend.State("some-restored-state")))
			Expect(linuxContainer.Events()).To(Equal([]string{
				"some-restored-event",
				"some-other-restored-event",
			}))
		})

		It("removes its UID from the pool", func() {
			_, err := pool.Restore(snapshot)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeUIDPool.Removed).To(ContainElement(uint32(10000)))
		})

		It("removes its network from the pool", func() {
			_, err := pool.Restore(snapshot)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeNetworkPool.Removed).To(ContainElement(restoredNetwork.String()))
		})

		It("removes its ports from the pool", func() {
			_, err := pool.Restore(snapshot)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakePortPool.Removed).To(ContainElement(uint32(61001)))
			Expect(fakePortPool.Removed).To(ContainElement(uint32(61002)))
			Expect(fakePortPool.Removed).To(ContainElement(uint32(61003)))
		})

		Context("when decoding the snapshot fails", func() {
			BeforeEach(func() {
				snapshot = new(bytes.Buffer)
			})

			It("fails", func() {
				_, err := pool.Restore(snapshot)
				Expect(err).To(HaveOccurred())
			})
		})

		Context("when removing the UID from the pool fails", func() {
			disaster := errors.New("oh no!")

			JustBeforeEach(func() {
				fakeUIDPool.RemoveError = disaster
			})

			It("returns the error", func() {
				_, err := pool.Restore(snapshot)
				Expect(err).To(Equal(disaster))
			})
		})

		Context("when removing the network from the pool fails", func() {
			disaster := errors.New("oh no!")

			JustBeforeEach(func() {
				fakeNetworkPool.RemoveError = disaster
			})

			It("returns the error and releases the uid", func() {
				_, err := pool.Restore(snapshot)
				Expect(err).To(Equal(disaster))

				Expect(fakeUIDPool.Released).To(ContainElement(uint32(10000)))
			})
		})

		Context("when removing a port from the pool fails", func() {
			disaster := errors.New("oh no!")

			JustBeforeEach(func() {
				fakePortPool.RemoveError = disaster
			})

			It("returns the error and releases the uid, network, and all ports", func() {
				_, err := pool.Restore(snapshot)
				Expect(err).To(Equal(disaster))

				Expect(fakeUIDPool.Released).To(ContainElement(uint32(10000)))
				Expect(fakeNetworkPool.Released).To(ContainElement(restoredNetwork.String()))
				Expect(fakePortPool.Released).To(ContainElement(uint32(61001)))
				Expect(fakePortPool.Released).To(ContainElement(uint32(61002)))
				Expect(fakePortPool.Released).To(ContainElement(uint32(61003)))
			})
		})
	})

	Describe("pruning", func() {
		It("destroys any containers that are not in the given map", func() {
			fakeRunner.WhenRunning(
				fake_command_runner.CommandSpec{
					Path: "ls",
					Args: []string{"/depot/path"},
				}, func(cmd *exec.Cmd) error {
					Expect(cmd.Stdout).ToNot(BeNil())

					cmd.Stdout.Write([]byte("container-1\n"))
					cmd.Stdout.Write([]byte("container-2\n"))
					cmd.Stdout.Write([]byte("tmp\n"))
					cmd.Stdout.Write([]byte("container-3\n"))

					return nil
				},
			)

			err := pool.Prune(map[string]bool{"container-2": true})
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeRunner).To(HaveExecutedSerially(
				fake_command_runner.CommandSpec{
					Path: "/root/path/destroy.sh",
					Args: []string{"/depot/path/container-1"},
				},
				fake_command_runner.CommandSpec{
					Path: "/root/path/destroy.sh",
					Args: []string{"/depot/path/container-3"},
				},
			))

			Expect(fakeRunner).ToNot(HaveExecutedSerially(
				fake_command_runner.CommandSpec{
					Path: "/root/path/destroy.sh",
					Args: []string{"/depot/path/container-2"},
				},
			))
		})

		Context("when ls fails", func() {
			disaster := errors.New("ls failed")

			BeforeEach(func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "ls",
					}, func(cmd *exec.Cmd) error {
						return disaster
					},
				)
			})

			It("returns the error", func() {
				err := pool.Prune(map[string]bool{})
				Expect(err).To(Equal(disaster))
			})
		})

		Context("when destroy.sh fails", func() {
			disaster := errors.New("destroy.sh failed")

			BeforeEach(func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "/root/path/destroy.sh",
					}, func(cmd *exec.Cmd) error {
						return disaster
					},
				)
			})

			It("returns the error", func() {
				fakeRunner.WhenRunning(
					fake_command_runner.CommandSpec{
						Path: "ls",
						Args: []string{"/depot/path"},
					}, func(cmd *exec.Cmd) error {
						Expect(cmd.Stdout).ToNot(BeNil())

						cmd.Stdout.Write([]byte("container-1\n"))
						cmd.Stdout.Write([]byte("container-2\n"))
						cmd.Stdout.Write([]byte("tmp\n"))
						cmd.Stdout.Write([]byte("container-3\n"))

						return nil
					},
				)

				err := pool.Prune(map[string]bool{})
				Expect(err).To(Equal(disaster))
			})
		})
	})

	Describe("destroying", func() {
		var createdContainer *linux_backend.LinuxContainer

		BeforeEach(func() {
			container, err := pool.Create(backend.ContainerSpec{})
			Expect(err).ToNot(HaveOccurred())

			createdContainer = container.(*linux_backend.LinuxContainer)

			createdContainer.Resources().AddPort(123)
			createdContainer.Resources().AddPort(456)
		})

		It("executes destroy.sh with the correct args and environment", func() {
			err := pool.Destroy(createdContainer)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeRunner).To(HaveExecutedSerially(
				fake_command_runner.CommandSpec{
					Path: "/root/path/destroy.sh",
					Args: []string{"/depot/path/" + createdContainer.ID()},
				},
			))
		})

		It("releases the container's ports, uid, and network", func() {
			err := pool.Destroy(createdContainer)
			Expect(err).ToNot(HaveOccurred())

			Expect(fakePortPool.Released).To(ContainElement(uint32(123)))
			Expect(fakePortPool.Released).To(ContainElement(uint32(456)))

			Expect(fakeUIDPool.Released).To(ContainElement(uint32(10000)))

			Expect(fakeNetworkPool.Released).To(ContainElement("1.2.0.0/30"))
		})
	})
})
