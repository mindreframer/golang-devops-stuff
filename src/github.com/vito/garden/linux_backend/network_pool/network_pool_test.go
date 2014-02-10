package network_pool_test

import (
	"net"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf-experimental/garden/linux_backend/network"
	"github.com/pivotal-cf-experimental/garden/linux_backend/network_pool"
)

var _ = Describe("Network Pool", func() {
	var pool *network_pool.RealNetworkPool

	BeforeEach(func() {
		_, ipNet, err := net.ParseCIDR("10.254.0.0/22")
		Expect(err).ToNot(HaveOccurred())

		pool = network_pool.New(ipNet)
	})

	Describe("acquiring", func() {
		It("takes the next network in the pool", func() {
			network1, err := pool.Acquire()
			Expect(err).ToNot(HaveOccurred())

			Expect(network1.String()).To(Equal("10.254.0.0/30"))

			network2, err := pool.Acquire()
			Expect(err).ToNot(HaveOccurred())

			Expect(network2.String()).To(Equal("10.254.0.4/30"))
		})

		Context("when the pool is exhausted", func() {
			It("returns an error", func() {
				for i := 0; i < 256; i++ {
					_, err := pool.Acquire()
					Expect(err).ToNot(HaveOccurred())
				}

				_, err := pool.Acquire()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("removing", func() {
		It("acquires a specific network from the pool", func() {
			_, ipNet, err := net.ParseCIDR("10.254.0.0/30")
			Expect(err).ToNot(HaveOccurred())

			err = pool.Remove(network.New(ipNet))
			Expect(err).ToNot(HaveOccurred())

			for i := 0; i < (256 - 1); i++ {
				network, err := pool.Acquire()
				Expect(err).ToNot(HaveOccurred())
				Expect(network.String()).ToNot(Equal("10.254.0.0/30"))
			}

			_, err = pool.Acquire()
			Expect(err).To(HaveOccurred())
		})

		Context("when the resource is already acquired", func() {
			It("returns a PortTakenError", func() {
				network, err := pool.Acquire()
				Expect(err).ToNot(HaveOccurred())

				err = pool.Remove(network)
				Expect(err).To(Equal(network_pool.NetworkTakenError{network}))
			})
		})
	})

	Describe("releasing", func() {
		It("places a network back and the end of the pool", func() {
			first, err := pool.Acquire()
			Expect(err).ToNot(HaveOccurred())

			pool.Release(first)

			for i := 0; i < 255; i++ {
				_, err := pool.Acquire()
				Expect(err).ToNot(HaveOccurred())
			}

			last, err := pool.Acquire()
			Expect(err).ToNot(HaveOccurred())
			Expect(last).To(Equal(first))
		})

		Context("when the released network is out of the range", func() {
			It("does not add it to the pool", func() {
				_, smallIPNet, err := net.ParseCIDR("10.255.0.0/32")
				Expect(err).ToNot(HaveOccurred())

				kiddiePool := network_pool.New(smallIPNet)

				_, err = kiddiePool.Acquire()
				Expect(err).ToNot(HaveOccurred())

				_, err = kiddiePool.Acquire()
				Expect(err).To(HaveOccurred())

				outOfRangeNetwork, err := pool.Acquire()
				Expect(err).ToNot(HaveOccurred())

				kiddiePool.Release(outOfRangeNetwork)

				_, err = kiddiePool.Acquire()
				Expect(err).To(HaveOccurred())
			})
		})
	})

	Describe("getting the network", func() {
		It("returns the network's *net.IPNet", func() {
			Expect(pool.Network().String()).To(Equal("10.254.0.0/22"))
		})
	})
})
