package cgroups_manager_test

import (
	"io/ioutil"
	"os"
	"path"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf-experimental/garden/linux_backend/cgroups_manager"
)

var _ = Describe("Container cgroups", func() {
	var cgroupsPath string
	var cgroupsManager *cgroups_manager.ContainerCgroupsManager

	BeforeEach(func() {
		tmpdir, err := ioutil.TempDir(os.TempDir(), "some-cgroups")
		Expect(err).ToNot(HaveOccurred())

		cgroupsPath = tmpdir

		cgroupsManager = cgroups_manager.New(cgroupsPath, "some-container-id")
	})

	Describe("setting", func() {
		It("writes the value to the name under the subsytem", func() {
			containerMemoryCgroupsPath := path.Join(cgroupsPath, "memory", "instance-some-container-id")
			err := os.MkdirAll(containerMemoryCgroupsPath, 0755)
			Expect(err).ToNot(HaveOccurred())

			err = cgroupsManager.Set("memory", "memory.limit_in_bytes", "42")
			Expect(err).ToNot(HaveOccurred())

			value, err := ioutil.ReadFile(path.Join(containerMemoryCgroupsPath, "memory.limit_in_bytes"))
			Expect(err).ToNot(HaveOccurred())
			Expect(string(value)).To(Equal("42"))
		})

		Context("when the cgroups directory does not exist", func() {
			BeforeEach(func() {
				err := os.RemoveAll(cgroupsPath)
				Expect(err).ToNot(HaveOccurred())
			})

			It("returns an error", func() {
				err := cgroupsManager.Set("memory", "memory.limit_in_bytes", "42")
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("getting", func() {
		It("reads the current value from the name under the subsystem", func() {
			containerMemoryCgroupsPath := path.Join(cgroupsPath, "memory", "instance-some-container-id")

			err := os.MkdirAll(containerMemoryCgroupsPath, 0755)
			Expect(err).ToNot(HaveOccurred())

			err = ioutil.WriteFile(path.Join(containerMemoryCgroupsPath, "memory.limit_in_bytes"), []byte("123\n"), 0644)
			Expect(err).ToNot(HaveOccurred())

			val, err := cgroupsManager.Get("memory", "memory.limit_in_bytes")
			Expect(err).ToNot(HaveOccurred())
			Expect(val).To(Equal("123"))
		})
	})

	Describe("retrieving a subsystem path", func() {
		It("returns <path>/<subsytem>/instance-<container-id>", func() {
			Expect(cgroupsManager.SubsystemPath("memory")).To(Equal(
				path.Join(cgroupsPath, "memory", "instance-some-container-id"),
			))
		})
	})
})
