package linux_backend_test

import (
	"errors"
	"io/ioutil"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf-experimental/garden/backend"
	"github.com/pivotal-cf-experimental/garden/backend/fake_backend"
	"github.com/pivotal-cf-experimental/garden/linux_backend"
	"github.com/pivotal-cf-experimental/garden/linux_backend/container_pool/fake_container_pool"
)

var _ = Describe("Setup", func() {
	var fakeContainerPool *fake_container_pool.FakeContainerPool
	var linuxBackend *linux_backend.LinuxBackend

	BeforeEach(func() {
		fakeContainerPool = fake_container_pool.New()
		linuxBackend = linux_backend.New(fakeContainerPool, "")
	})

	It("sets up the container pool", func() {
		err := linuxBackend.Setup()
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeContainerPool.DidSetup).To(BeTrue())
	})
})

var _ = Describe("Start", func() {
	var fakeContainerPool *fake_container_pool.FakeContainerPool

	var tmpdir string

	BeforeEach(func() {
		var err error

		tmpdir, err = ioutil.TempDir(os.TempDir(), "warden-server-test")
		Expect(err).ToNot(HaveOccurred())

		fakeContainerPool = fake_container_pool.New()
	})

	It("creates the snapshots directory if it's not already there", func() {
		snapshotsPath := path.Join(tmpdir, "snapshots")

		linuxBackend := linux_backend.New(fakeContainerPool, snapshotsPath)

		err := linuxBackend.Start()
		Expect(err).ToNot(HaveOccurred())

		stat, err := os.Stat(snapshotsPath)
		Expect(err).ToNot(HaveOccurred())

		Expect(stat.IsDir()).To(BeTrue())
	})

	Context("when the snapshots directory fails to be created", func() {
		It("fails to start", func() {
			tmpfile, err := ioutil.TempFile(os.TempDir(), "warden-server-test")
			Expect(err).ToNot(HaveOccurred())

			linuxBackend := linux_backend.New(
				fakeContainerPool,
				// weird scenario: /foo/X/snapshots with X being a file
				path.Join(tmpfile.Name(), "snapshots"),
			)

			err = linuxBackend.Start()
			Expect(err).To(HaveOccurred())
		})
	})

	Context("when no snapshots directory is given", func() {
		It("successfully starts", func() {
			linuxBackend := linux_backend.New(fakeContainerPool, "")

			err := linuxBackend.Start()
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Describe("when snapshots are present", func() {
		var snapshotsPath string

		BeforeEach(func() {
			snapshotsPath = path.Join(tmpdir, "snapshots")

			err := os.MkdirAll(snapshotsPath, 0755)
			Expect(err).ToNot(HaveOccurred())

			file, err := os.Create(path.Join(snapshotsPath, "some-id"))
			Expect(err).ToNot(HaveOccurred())

			file.Write([]byte("handle-a"))
			file.Close()

			file, err = os.Create(path.Join(snapshotsPath, "some-other-id"))
			Expect(err).ToNot(HaveOccurred())

			file.Write([]byte("handle-b"))
			file.Close()
		})

		It("restores them via the container pool", func() {
			linuxBackend := linux_backend.New(fakeContainerPool, snapshotsPath)

			Expect(fakeContainerPool.RestoredSnapshots).To(BeEmpty())

			err := linuxBackend.Start()
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeContainerPool.RestoredSnapshots).To(HaveLen(2))
		})

		It("removes the snapshots", func() {
			linuxBackend := linux_backend.New(fakeContainerPool, snapshotsPath)

			Expect(fakeContainerPool.RestoredSnapshots).To(BeEmpty())

			err := linuxBackend.Start()
			Expect(err).ToNot(HaveOccurred())

			_, err = os.Stat(path.Join(snapshotsPath, "some-id"))
			Expect(err).To(HaveOccurred())

			_, err = os.Stat(path.Join(snapshotsPath, "some-other-id"))
			Expect(err).To(HaveOccurred())
		})

		It("registers the containers", func() {
			linuxBackend := linux_backend.New(fakeContainerPool, snapshotsPath)

			err := linuxBackend.Start()
			Expect(err).ToNot(HaveOccurred())

			containers, err := linuxBackend.Containers()
			Expect(err).ToNot(HaveOccurred())

			Expect(containers).To(HaveLen(2))
		})

		It("keeps them when pruning the container pool", func() {
			linuxBackend := linux_backend.New(fakeContainerPool, snapshotsPath)

			err := linuxBackend.Start()
			Expect(err).ToNot(HaveOccurred())

			Expect(fakeContainerPool.Pruned).To(BeTrue())
			Expect(fakeContainerPool.KeptContainers).To(Equal(map[string]bool{
				"handle-a": true,
				"handle-b": true,
			}))
		})

		Context("when restoring the container fails", func() {
			disaster := errors.New("failed to restore")

			BeforeEach(func() {
				fakeContainerPool.RestoreError = disaster
			})

			It("returns the error", func() {
				linuxBackend := linux_backend.New(fakeContainerPool, snapshotsPath)

				err := linuxBackend.Start()
				Expect(err).To(Equal(disaster))
			})
		})
	})

	It("prunes the container pool", func() {
		linuxBackend := linux_backend.New(fakeContainerPool, "")

		err := linuxBackend.Start()
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeContainerPool.Pruned).To(BeTrue())
		Expect(fakeContainerPool.KeptContainers).To(Equal(map[string]bool{}))
	})

	Context("when pruning the container pool fails", func() {
		disaster := errors.New("failed to prune")

		BeforeEach(func() {
			fakeContainerPool.PruneError = disaster
		})

		It("returns the error", func() {
			linuxBackend := linux_backend.New(fakeContainerPool, "")

			err := linuxBackend.Start()
			Expect(err).To(Equal(disaster))
		})
	})
})

var _ = Describe("Stop", func() {
	var fakeContainerPool *fake_container_pool.FakeContainerPool
	var linuxBackend *linux_backend.LinuxBackend

	BeforeEach(func() {
		tmpdir, err := ioutil.TempDir(os.TempDir(), "warden-server-test")
		Expect(err).ToNot(HaveOccurred())

		fakeContainerPool = fake_container_pool.New()
		linuxBackend = linux_backend.New(
			fakeContainerPool,
			path.Join(tmpdir, "snapshots"),
		)
	})

	It("takes a snapshot of each container", func() {
		container1, err := linuxBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
		Expect(err).ToNot(HaveOccurred())

		container2, err := linuxBackend.Create(backend.ContainerSpec{Handle: "some-other-handle"})
		Expect(err).ToNot(HaveOccurred())

		linuxBackend.Stop()

		fakeContainer1 := container1.(*fake_backend.FakeContainer)
		fakeContainer2 := container2.(*fake_backend.FakeContainer)
		Expect(fakeContainer1.SavedSnapshots).To(HaveLen(1))
		Expect(fakeContainer2.SavedSnapshots).To(HaveLen(1))
	})

	It("cleans up each container", func() {
		container1, err := linuxBackend.Create(backend.ContainerSpec{Handle: "some-handle"})
		Expect(err).ToNot(HaveOccurred())

		container2, err := linuxBackend.Create(backend.ContainerSpec{Handle: "some-other-handle"})
		Expect(err).ToNot(HaveOccurred())

		linuxBackend.Stop()

		fakeContainer1 := container1.(*fake_backend.FakeContainer)
		fakeContainer2 := container2.(*fake_backend.FakeContainer)
		Expect(fakeContainer1.CleanedUp).To(BeTrue())
		Expect(fakeContainer2.CleanedUp).To(BeTrue())
	})
})

var _ = Describe("Create", func() {
	var fakeContainerPool *fake_container_pool.FakeContainerPool
	var linuxBackend *linux_backend.LinuxBackend

	BeforeEach(func() {
		fakeContainerPool = fake_container_pool.New()
		linuxBackend = linux_backend.New(fakeContainerPool, "")
	})

	It("creates a container from the pool", func() {
		Expect(fakeContainerPool.CreatedContainers).To(BeEmpty())

		container, err := linuxBackend.Create(backend.ContainerSpec{})
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeContainerPool.CreatedContainers).To(ContainElement(container))
	})

	It("starts the container", func() {
		container, err := linuxBackend.Create(backend.ContainerSpec{})
		Expect(err).ToNot(HaveOccurred())
		Expect(container.(*fake_backend.FakeContainer).Started).To(BeTrue())
	})

	It("registers the container", func() {
		container, err := linuxBackend.Create(backend.ContainerSpec{})
		Expect(err).ToNot(HaveOccurred())

		foundContainer, err := linuxBackend.Lookup(container.Handle())
		Expect(err).ToNot(HaveOccurred())

		Expect(foundContainer).To(Equal(container))
	})

	Context("when creating the container fails", func() {
		disaster := errors.New("failed to create")

		BeforeEach(func() {
			fakeContainerPool.CreateError = disaster
		})

		It("returns the error", func() {
			container, err := linuxBackend.Create(backend.ContainerSpec{})
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(disaster))

			Expect(container).To(BeNil())
		})
	})

	Context("when starting the container fails", func() {
		disaster := errors.New("failed to start")

		BeforeEach(func() {
			fakeContainerPool.ContainerSetup = func(c *fake_backend.FakeContainer) {
				c.StartError = disaster
			}
		})

		It("returns the error", func() {
			container, err := linuxBackend.Create(backend.ContainerSpec{})
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(disaster))

			Expect(container).To(BeNil())
		})

		It("does not register the container", func() {
			_, err := linuxBackend.Create(backend.ContainerSpec{})
			Expect(err).To(HaveOccurred())

			containers, err := linuxBackend.Containers()
			Expect(err).ToNot(HaveOccurred())

			Expect(containers).To(BeEmpty())
		})
	})
})

var _ = Describe("Destroy", func() {
	var fakeContainerPool *fake_container_pool.FakeContainerPool
	var linuxBackend *linux_backend.LinuxBackend

	var container backend.Container

	BeforeEach(func() {
		fakeContainerPool = fake_container_pool.New()
		linuxBackend = linux_backend.New(fakeContainerPool, "")

		newContainer, err := linuxBackend.Create(backend.ContainerSpec{})
		Expect(err).ToNot(HaveOccurred())

		container = newContainer
	})

	It("removes the given container from the pool", func() {
		Expect(fakeContainerPool.DestroyedContainers).To(BeEmpty())

		err := linuxBackend.Destroy(container.Handle())
		Expect(err).ToNot(HaveOccurred())

		Expect(fakeContainerPool.DestroyedContainers).To(ContainElement(container))
	})

	It("unregisters the container", func() {
		err := linuxBackend.Destroy(container.Handle())
		Expect(err).ToNot(HaveOccurred())

		_, err = linuxBackend.Lookup(container.Handle())
		Expect(err).To(HaveOccurred())
		Expect(err).To(Equal(linux_backend.UnknownHandleError{container.Handle()}))
	})

	Context("when the container does not exist", func() {
		It("returns UnknownHandleError", func() {
			err := linuxBackend.Destroy("bogus-handle")
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(linux_backend.UnknownHandleError{"bogus-handle"}))
		})
	})

	Context("when destroying the container fails", func() {
		disaster := errors.New("failed to destroy")

		BeforeEach(func() {
			fakeContainerPool.DestroyError = disaster
		})

		It("returns the error", func() {
			err := linuxBackend.Destroy(container.Handle())
			Expect(err).To(HaveOccurred())
			Expect(err).To(Equal(disaster))
		})

		It("does not unregister the container", func() {
			err := linuxBackend.Destroy(container.Handle())
			Expect(err).To(HaveOccurred())

			foundContainer, err := linuxBackend.Lookup(container.Handle())
			Expect(err).ToNot(HaveOccurred())
			Expect(foundContainer).To(Equal(container))
		})
	})
})

var _ = Describe("Lookup", func() {
	var fakeContainerPool *fake_container_pool.FakeContainerPool
	var linuxBackend *linux_backend.LinuxBackend

	BeforeEach(func() {
		fakeContainerPool = fake_container_pool.New()
		linuxBackend = linux_backend.New(fakeContainerPool, "")
	})

	It("returns the container", func() {
		container, err := linuxBackend.Create(backend.ContainerSpec{})
		Expect(err).ToNot(HaveOccurred())

		foundContainer, err := linuxBackend.Lookup(container.Handle())
		Expect(err).ToNot(HaveOccurred())

		Expect(foundContainer).To(Equal(container))
	})

	Context("when the handle is not found", func() {
		It("returns UnknownHandleError", func() {
			foundContainer, err := linuxBackend.Lookup("bogus-handle")
			Expect(err).To(HaveOccurred())
			Expect(foundContainer).To(BeNil())

			Expect(err).To(Equal(linux_backend.UnknownHandleError{"bogus-handle"}))
		})
	})
})

var _ = Describe("Containers", func() {
	var fakeContainerPool *fake_container_pool.FakeContainerPool
	var linuxBackend *linux_backend.LinuxBackend

	BeforeEach(func() {
		fakeContainerPool = fake_container_pool.New()
		linuxBackend = linux_backend.New(fakeContainerPool, "")
	})

	It("returns a list of all existing containers", func() {
		container1, err := linuxBackend.Create(backend.ContainerSpec{})
		Expect(err).ToNot(HaveOccurred())

		container2, err := linuxBackend.Create(backend.ContainerSpec{})
		Expect(err).ToNot(HaveOccurred())

		containers, err := linuxBackend.Containers()
		Expect(err).ToNot(HaveOccurred())

		Expect(containers).To(ContainElement(container1))
		Expect(containers).To(ContainElement(container2))
	})
})
