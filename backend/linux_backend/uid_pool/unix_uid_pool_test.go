package uid_pool_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vito/garden/backend/linux_backend/uid_pool"
)

var _ = Describe("Unix UID pool", func() {
	Describe("acquiring", func() {
		It("returns the next available UID from the pool", func() {
			pool := uid_pool.New(10000, 5)

			uid1, err := pool.Acquire()
			Expect(err).ToNot(HaveOccured())

			uid2, err := pool.Acquire()
			Expect(err).ToNot(HaveOccured())

			Expect(uid1).To(Equal(uint32(10000)))
			Expect(uid2).To(Equal(uint32(10001)))
		})

		Context("when the pool is exhausted", func() {
			It("returns an error", func() {
				pool := uid_pool.New(10000, 5)

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
		It("places a uid back at the end of the pool", func() {
			pool := uid_pool.New(10000, 2)

			uid1, err := pool.Acquire()
			Expect(err).ToNot(HaveOccured())
			Expect(uid1).To(Equal(uint32(10000)))

			pool.Release(uid1)

			uid2, err := pool.Acquire()
			Expect(err).ToNot(HaveOccured())
			Expect(uid2).To(Equal(uint32(10001)))

			nextUID, err := pool.Acquire()
			Expect(err).ToNot(HaveOccured())
			Expect(nextUID).To(Equal(uint32(10000)))
		})

		Context("when the released uid is out of the range", func() {
			It("does not add it to the pool", func() {
				pool := uid_pool.New(10000, 0)

				pool.Release(20000)

				_, err := pool.Acquire()
				Expect(err).To(HaveOccured())
			})
		})
	})
})
