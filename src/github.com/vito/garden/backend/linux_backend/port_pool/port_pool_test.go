package port_pool_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vito/garden/backend/linux_backend/port_pool"
)

var _ = Describe("Port pool", func() {
	Describe("acquiring", func() {
		It("returns the next available port from the pool", func() {
			pool := port_pool.New(10000, 5)

			port1, err := pool.Acquire()
			Expect(err).ToNot(HaveOccured())

			port2, err := pool.Acquire()
			Expect(err).ToNot(HaveOccured())

			Expect(port1).To(Equal(uint32(10000)))
			Expect(port2).To(Equal(uint32(10001)))
		})

		Context("when the pool is exhausted", func() {
			It("returns an error", func() {
				pool := port_pool.New(10000, 5)

				for i := 0; i < 5; i++ {
					_, err := pool.Acquire()
					Expect(err).ToNot(HaveOccured())
				}

				_, err := pool.Acquire()
				Expect(err).To(HaveOccured())
			})
		})
	})

	Describe("releasing", func() {
		It("places a port back at the end of the pool", func() {
			pool := port_pool.New(10000, 2)

			port1, err := pool.Acquire()
			Expect(err).ToNot(HaveOccured())
			Expect(port1).To(Equal(uint32(10000)))

			pool.Release(port1)

			port2, err := pool.Acquire()
			Expect(err).ToNot(HaveOccured())
			Expect(port2).To(Equal(uint32(10001)))

			nextPort, err := pool.Acquire()
			Expect(err).ToNot(HaveOccured())
			Expect(nextPort).To(Equal(uint32(10000)))
		})

		Context("when the released port is out of the range", func() {
			It("does not add it to the pool", func() {
				pool := port_pool.New(10000, 0)

				pool.Release(20000)

				_, err := pool.Acquire()
				Expect(err).To(HaveOccured())
			})
		})
	})
})
