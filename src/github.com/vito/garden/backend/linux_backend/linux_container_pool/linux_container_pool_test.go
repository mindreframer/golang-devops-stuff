package linux_container_pool_test

import (
	"errors"
	"net"
	"os/exec"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vito/garden/backend"
	"github.com/vito/garden/backend/linux_backend"
	"github.com/vito/garden/backend/linux_backend/linux_container_pool"
	"github.com/vito/garden/backend/linux_backend/network_pool/fake_network_pool"
	"github.com/vito/garden/backend/linux_backend/port_pool/fake_port_pool"
	"github.com/vito/garden/backend/linux_backend/quota_manager/fake_quota_manager"
	"github.com/vito/garden/backend/linux_backend/uid_pool/fake_uid_pool"
	"github.com/vito/garden/command_runner/fake_command_runner"
	. "github.com/vito/garden/command_runner/fake_command_runner/matchers"
)

var _ = Describe("Linux Container pool", func() {
	var fakeRunner *fake_command_runner.FakeCommandRunner
	var fakeUIDPool *fake_uid_pool.FakeUIDPool
	var fakeNetworkPool *fake_network_pool.FakeNetworkPool
	var fakeQuotaManager *fake_quota_manager.FakeQuotaManager
	var fakePortPool *fake_port_pool.FakePortPool
	var pool *linux_container_pool.LinuxContainerPool

	BeforeEach(func() {
		_, ipNet, err := net.ParseCIDR("1.2.0.0/20")
		Expect(err).ToNot(HaveOccured())

		fakeUIDPool = fake_uid_pool.New(10000)

		fakeNetworkPool = fake_network_pool.New(ipNet)
		fakeRunner = fake_command_runner.New()
		fakeQuotaManager = fake_quota_manager.New()
		fakePortPool = fake_port_pool.New(1000)

		pool = linux_container_pool.New(
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
			Expect(err).ToNot(HaveOccured())

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
			Expect(err).ToNot(HaveOccured())

			container2, err := pool.Create(backend.ContainerSpec{})
			Expect(err).ToNot(HaveOccured())

			Expect(container1.ID()).ToNot(Equal(container2.ID()))
		})

		It("executes create.sh with the correct args and environment", func() {
			container, err := pool.Create(backend.ContainerSpec{})
			Expect(err).ToNot(HaveOccured())

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

				Expect(err).ToNot(HaveOccured())

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

	Describe("destroying", func() {
		var createdContainer *linux_backend.LinuxContainer

		BeforeEach(func() {
			container, err := pool.Create(backend.ContainerSpec{})
			Expect(err).ToNot(HaveOccured())

			createdContainer = container.(*linux_backend.LinuxContainer)

			createdContainer.Resources().AddPort(123)
			createdContainer.Resources().AddPort(456)
		})

		It("executes destroy.sh with the correct args and environment", func() {
			err := pool.Destroy(createdContainer)
			Expect(err).ToNot(HaveOccured())

			Expect(fakeRunner).To(HaveExecutedSerially(
				fake_command_runner.CommandSpec{
					Path: "/root/path/destroy.sh",
					Args: []string{"/depot/path/" + createdContainer.ID()},
					Env: []string{
						"id=" + createdContainer.ID(),
						"PATH=/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin",
					},
				},
			))
		})

		It("releases the container's ports, uid, and network", func() {
			err := pool.Destroy(createdContainer)
			Expect(err).ToNot(HaveOccured())

			Expect(fakePortPool.Released).To(ContainElement(uint32(123)))
			Expect(fakePortPool.Released).To(ContainElement(uint32(456)))

			Expect(fakeUIDPool.Released).To(ContainElement(uint32(10000)))

			Expect(fakeNetworkPool.Released).To(ContainElement("1.2.0.0/30"))
		})
	})
})
