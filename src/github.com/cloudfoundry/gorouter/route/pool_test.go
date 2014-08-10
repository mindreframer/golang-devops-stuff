package route_test

import (
	. "github.com/cloudfoundry/gorouter/route"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"time"
)

var _ = Describe("Pool", func() {
	var pool *Pool

	BeforeEach(func() {
		pool = NewPool(2 * time.Minute)
	})

	Context("Put", func() {
		It("adds endpoints", func() {
			endpoint := &Endpoint{}

			b := pool.Put(endpoint)
			Ω(b).Should(BeTrue())
		})

		It("handles duplicate endpoints", func() {
			endpoint := &Endpoint{}

			pool.Put(endpoint)
			b := pool.Put(endpoint)
			Ω(b).Should(BeFalse())
		})

		It("handles equivalent (duplicate) endpoints", func() {
			endpoint1 := NewEndpoint("", "1.2.3.4", 5678, "", nil)
			endpoint2 := NewEndpoint("", "1.2.3.4", 5678, "", nil)

			pool.Put(endpoint1)
			Ω(pool.Put(endpoint2)).Should(BeFalse())
		})
	})

	Context("Remove", func() {
		It("removes endpoints", func() {
			endpoint := &Endpoint{}
			pool.Put(endpoint)

			b := pool.Remove(endpoint)
			Ω(b).Should(BeTrue())
			Ω(pool.IsEmpty()).Should(BeTrue())
		})

		It("fails to remove an endpoint that doesn't exist", func() {
			endpoint := &Endpoint{}

			b := pool.Remove(endpoint)
			Ω(b).Should(BeFalse())
		})
	})

	Context("IsEmpty", func() {
		It("starts empty", func() {
			Ω(pool.IsEmpty()).To(BeTrue())
		})

		It("not empty after adding an endpoint", func() {
			endpoint := &Endpoint{}
			pool.Put(endpoint)

			Ω(pool.IsEmpty()).Should(BeFalse())
		})

		It("is empty after removing everything", func() {
			endpoint := &Endpoint{}
			pool.Put(endpoint)
			pool.Remove(endpoint)

			Ω(pool.IsEmpty()).To(BeTrue())
		})
	})

	Context("PruneBefore", func() {
		It("prunes endpoints that haven't been updated", func() {
			e1 := NewEndpoint("", "1.2.3.4", 5678, "", nil)
			e2 := NewEndpoint("", "5.6.7.8", 1234, "", nil)
			pool.Put(e1)
			pool.Put(e2)

			t := time.Now().Add(1 * time.Second)
			pool.PruneBefore(t)
			Ω(pool.IsEmpty()).Should(BeTrue())
		})

		It("does not prune updated endpoints", func() {
			e1 := NewEndpoint("", "1.2.3.4", 5678, "", nil)
			e2 := NewEndpoint("", "5.6.7.8", 1234, "", nil)
			pool.Put(e1)
			pool.Put(e2)

			t := time.Now().Add(-1 * time.Second)
			pool.PruneBefore(t)
			Ω(pool.IsEmpty()).Should(BeFalse())

			iter := pool.Endpoints("")
			n1 := iter.Next()
			n2 := iter.Next()
			Ω(n1).ShouldNot(Equal(n2))
		})
	})

	Context("MarkUpdated", func() {
		It("updates all endpoints", func() {
			e1 := NewEndpoint("", "1.2.3.4", 5678, "", nil)

			pool.Put(e1)

			t := time.Time{}.Add(1 * time.Second)
			pool.PruneBefore(t)
			Ω(pool.IsEmpty()).Should(BeFalse())

			pool.MarkUpdated(t)
			pool.PruneBefore(t)
			Ω(pool.IsEmpty()).Should(BeFalse())

			pool.PruneBefore(t.Add(1 * time.Microsecond))
			Ω(pool.IsEmpty()).Should(BeTrue())
		})
	})

	Context("Each", func() {
		It("applies a function to each endpoint", func() {
			e1 := NewEndpoint("", "1.2.3.4", 5678, "", nil)
			e2 := NewEndpoint("", "5.6.7.8", 1234, "", nil)
			pool.Put(e1)
			pool.Put(e2)

			endpoints := make(map[string]*Endpoint)
			pool.Each(func(e *Endpoint) {
				endpoints[e.CanonicalAddr()] = e
			})
			Ω(endpoints).Should(HaveLen(2))
			Ω(endpoints[e1.CanonicalAddr()]).Should(Equal(e1))
			Ω(endpoints[e2.CanonicalAddr()]).Should(Equal(e2))
		})
	})

	It("marshals json", func() {
		e := NewEndpoint("", "1.2.3.4", 5678, "", nil)
		pool.Put(e)

		json, err := pool.MarshalJSON()
		Ω(err).ToNot(HaveOccurred())

		Ω(string(json)).To(Equal(`["1.2.3.4:5678"]`))
	})
})
