package linux_backend_test

import (
	"errors"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vito/garden/backend"
	"github.com/vito/garden/backend/fake_backend"
	"github.com/vito/garden/backend/fake_backend/fake_container_pool"
	"github.com/vito/garden/backend/linux_backend"
)

var fakeContainerPool *fake_container_pool.FakeContainerPool
var linuxBackend *linux_backend.LinuxBackend

var _ = Describe("Setup", func() {
	BeforeEach(func() {
		fakeContainerPool = fake_container_pool.New()
		linuxBackend = linux_backend.New(fakeContainerPool)
	})

	It("sets up the container pool", func() {
		err := linuxBackend.Setup()
		Expect(err).ToNot(HaveOccured())

		Expect(fakeContainerPool.DidSetup).To(BeTrue())
	})
})

var _ = Describe("Create", func() {
	BeforeEach(func() {
		fakeContainerPool = fake_container_pool.New()
		linuxBackend = linux_backend.New(fakeContainerPool)
	})

	It("creates a container from the pool", func() {
		Expect(fakeContainerPool.CreatedContainers).To(BeEmpty())

		container, err := linuxBackend.Create(backend.ContainerSpec{})
		Expect(err).ToNot(HaveOccured())

		Expect(fakeContainerPool.CreatedContainers).To(ContainElement(container))
	})

	It("starts the container", func() {
		container, err := linuxBackend.Create(backend.ContainerSpec{})
		Expect(err).ToNot(HaveOccured())
		Expect(container.(*fake_backend.FakeContainer).Started).To(BeTrue())
	})

	It("registers the container", func() {
		container, err := linuxBackend.Create(backend.ContainerSpec{})
		Expect(err).ToNot(HaveOccured())

		foundContainer, err := linuxBackend.Lookup(container.Handle())
		Expect(err).ToNot(HaveOccured())

		Expect(foundContainer).To(Equal(container))
	})

	Context("when creating the container fails", func() {
		disaster := errors.New("oh no!")

		BeforeEach(func() {
			fakeContainerPool.CreateError = disaster
		})

		It("returns the error", func() {
			container, err := linuxBackend.Create(backend.ContainerSpec{})
			Expect(err).To(HaveOccured())
			Expect(err).To(Equal(disaster))

			Expect(container).To(BeNil())
		})
	})

	Context("when starting the container fails", func() {
		disaster := errors.New("oh no!")

		BeforeEach(func() {
			fakeContainerPool.ContainerSetup = func(c *fake_backend.FakeContainer) {
				c.StartError = disaster
			}
		})

		It("returns the error", func() {
			container, err := linuxBackend.Create(backend.ContainerSpec{})
			Expect(err).To(HaveOccured())
			Expect(err).To(Equal(disaster))

			Expect(container).To(BeNil())
		})

		It("does not register the container", func() {
			_, err := linuxBackend.Create(backend.ContainerSpec{})
			Expect(err).To(HaveOccured())

			containers, err := linuxBackend.Containers()
			Expect(err).ToNot(HaveOccured())

			Expect(containers).To(BeEmpty())
		})
	})
})

var _ = Describe("Destroy", func() {
	var container backend.Container

	BeforeEach(func() {
		fakeContainerPool = fake_container_pool.New()
		linuxBackend = linux_backend.New(fakeContainerPool)

		newContainer, err := linuxBackend.Create(backend.ContainerSpec{})
		Expect(err).ToNot(HaveOccured())

		container = newContainer
	})

	It("removes the given container from the pool", func() {
		Expect(fakeContainerPool.DestroyedContainers).To(BeEmpty())

		err := linuxBackend.Destroy(container.Handle())
		Expect(err).ToNot(HaveOccured())

		Expect(fakeContainerPool.DestroyedContainers).To(ContainElement(container))
	})

	It("unregisters the container", func() {
		err := linuxBackend.Destroy(container.Handle())
		Expect(err).ToNot(HaveOccured())

		_, err = linuxBackend.Lookup(container.Handle())
		Expect(err).To(HaveOccured())
		Expect(err).To(Equal(linux_backend.UnknownHandleError{container.Handle()}))
	})

	Context("when the container does not exist", func() {
		It("returns UnknownHandleError", func() {
			err := linuxBackend.Destroy("bogus-handle")
			Expect(err).To(HaveOccured())
			Expect(err).To(Equal(linux_backend.UnknownHandleError{"bogus-handle"}))
		})
	})

	Context("when destroying the container fails", func() {
		disaster := errors.New("oh no!")

		BeforeEach(func() {
			fakeContainerPool.DestroyError = disaster
		})

		It("returns the error", func() {
			err := linuxBackend.Destroy(container.Handle())
			Expect(err).To(HaveOccured())
			Expect(err).To(Equal(disaster))
		})

		It("does not unregister the container", func() {
			err := linuxBackend.Destroy(container.Handle())
			Expect(err).To(HaveOccured())

			foundContainer, err := linuxBackend.Lookup(container.Handle())
			Expect(err).ToNot(HaveOccured())
			Expect(foundContainer).To(Equal(container))
		})
	})
})

var _ = Describe("Lookup", func() {
	BeforeEach(func() {
		fakeContainerPool = fake_container_pool.New()
		linuxBackend = linux_backend.New(fakeContainerPool)
	})

	It("returns the container", func() {
		container, err := linuxBackend.Create(backend.ContainerSpec{})
		Expect(err).ToNot(HaveOccured())

		foundContainer, err := linuxBackend.Lookup(container.Handle())
		Expect(err).ToNot(HaveOccured())

		Expect(foundContainer).To(Equal(container))
	})

	Context("when the handle is not found", func() {
		It("returns UnknownHandleError", func() {
			foundContainer, err := linuxBackend.Lookup("bogus-handle")
			Expect(err).To(HaveOccured())
			Expect(foundContainer).To(BeNil())

			Expect(err).To(Equal(linux_backend.UnknownHandleError{"bogus-handle"}))
		})
	})
})

var _ = Describe("Containers", func() {
	BeforeEach(func() {
		fakeContainerPool = fake_container_pool.New()
		linuxBackend = linux_backend.New(fakeContainerPool)
	})

	It("returns a list of all existing containers", func() {
		container1, err := linuxBackend.Create(backend.ContainerSpec{})
		Expect(err).ToNot(HaveOccured())

		container2, err := linuxBackend.Create(backend.ContainerSpec{})
		Expect(err).ToNot(HaveOccured())

		containers, err := linuxBackend.Containers()
		Expect(err).ToNot(HaveOccured())

		Expect(containers).To(ContainElement(container1))
		Expect(containers).To(ContainElement(container2))
	})
})
