package lifecycle_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Creating a container", func() {
	var handle string

	BeforeEach(func() {
		res, err := client.Create()
		Expect(err).ToNot(HaveOccurred())

		handle = res.GetHandle()
	})

	AfterEach(func() {
		_, err := client.Destroy(handle)
		Expect(err).ToNot(HaveOccurred())
	})

	Context("and sending a List request", func() {
		It("includes the created container", func() {
			res, err := client.List()
			Expect(err).ToNot(HaveOccurred())

			Expect(res.GetHandles()).To(ContainElement(handle))
		})
	})

	Context("and sending an Info request", func() {
		It("returns the container's info", func() {
			res, err := client.Info(handle)
			Expect(err).ToNot(HaveOccurred())

			Expect(res.GetState()).To(Equal("active"))
		})
	})

	Context("and starting and streaming a job", func() {
		It("sends output back in chunks until stopped", func() {
			res, err := client.Spawn(
				handle,
				"sleep 0.5; echo hello; sleep 0.5; echo goodbye; sleep 0.5; exit 42",
				true,
			)

			Expect(err).ToNot(HaveOccurred())

			stream, err := client.Stream(handle, res.GetJobId())

			Expect((<-stream).GetData()).To(Equal("hello\n"))
			Expect((<-stream).GetData()).To(Equal("goodbye\n"))
			Expect((<-stream).GetExitStatus()).To(Equal(uint32(42)))
		})

		Context("and then sending a Stop request", func() {
			It("kills the job", func(done Done) {
				res, err := client.Spawn(
					handle,
					`exec ruby -e 'trap("TERM") { exit 42 }; while true; sleep 1; end'`,
					true,
				)

				Expect(err).ToNot(HaveOccurred())

				stream, err := client.Stream(handle, res.GetJobId())

				_, err = client.Stop(handle, false, false)
				Expect(err).ToNot(HaveOccurred())

				Expect((<-stream).GetExitStatus()).To(Equal(uint32(42)))

				close(done)
			}, 10.0)
		})
	})

	Context("and sending a Stop request", func() {
		It("changes the container's state to 'stopped'", func() {
			_, err := client.Stop(handle, false, false)
			Expect(err).ToNot(HaveOccurred())

			info, err := client.Info(handle)
			Expect(err).ToNot(HaveOccurred())

			Expect(info.GetState()).To(Equal("stopped"))
		})
	})
})
