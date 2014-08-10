package route_test

import (
	"time"
	. "github.com/cloudfoundry/gorouter/route"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("EndpointIterator", func() {
	var pool *Pool

	BeforeEach(func() {
		pool = NewPool(2 * time.Minute)
	})

	Describe("Next", func() {
		It("performs round-robin through the endpoints", func() {
			e1 := NewEndpoint("", "1.2.3.4", 5678, "", nil)
			e2 := NewEndpoint("", "5.6.7.8", 1234, "", nil)
			e3 := NewEndpoint("", "1.2.7.8", 1234, "", nil)
			endpoints := []*Endpoint{e1, e2, e3}

			for _, e := range endpoints {
				pool.Put(e)
			}

			counts := make([]int, len(endpoints))

			iter := pool.Endpoints("")

			loops := 50
			for i := 0; i < len(endpoints)*loops; i += 1 {
				n := iter.Next()
				for j, e := range endpoints {
					if e == n {
						counts[j]++
						break
					}
				}
			}

			for i := 0; i < len(endpoints); i++ {
				Ω(counts[i]).To(Equal(loops))
			}
		})

		It("returns nil when no endpoints exist", func() {
			iter := pool.Endpoints("")
			e := iter.Next()
			Ω(e).Should(BeNil())
		})

		It("finds the initial endpoint by private id", func() {
			b := NewEndpoint("", "1.2.3.4", 1235, "b", nil)
			pool.Put(NewEndpoint("", "1.2.3.4", 1234, "a", nil))
			pool.Put(b)
			pool.Put(NewEndpoint("", "1.2.3.4", 1236, "c", nil))
			pool.Put(NewEndpoint("", "1.2.3.4", 1237, "d", nil))

			for i := 0; i < 10; i++ {
				iter := pool.Endpoints(b.PrivateInstanceId)
				e := iter.Next()
				Ω(e).ShouldNot(BeNil())
				Ω(e.PrivateInstanceId).To(Equal(b.PrivateInstanceId))
			}
		})

		It("finds the initial endpoint by canonical addr", func() {
			b := NewEndpoint("", "1.2.3.4", 1235, "b", nil)
			pool.Put(NewEndpoint("", "1.2.3.4", 1234, "a", nil))
			pool.Put(b)
			pool.Put(NewEndpoint("", "1.2.3.4", 1236, "c", nil))
			pool.Put(NewEndpoint("", "1.2.3.4", 1237, "d", nil))

			for i := 0; i < 10; i++ {
				iter := pool.Endpoints(b.CanonicalAddr())
				e := iter.Next()
				Ω(e).ShouldNot(BeNil())
				Ω(e.CanonicalAddr()).To(Equal(b.CanonicalAddr()))
			}
		})

		It("finds when there are multiple private ids", func() {
			endpointFoo := NewEndpoint("", "1.2.3.4", 1234, "foo", nil)
			endpointBar := NewEndpoint("", "5.6.7.8", 5678, "bar", nil)

			pool.Put(endpointFoo)
			pool.Put(endpointBar)

			iter := pool.Endpoints(endpointFoo.PrivateInstanceId)
			foundEndpoint := iter.Next()
			Ω(foundEndpoint).ToNot(BeNil())
			Ω(foundEndpoint).To(Equal(endpointFoo))

			iter = pool.Endpoints(endpointBar.PrivateInstanceId)
			foundEndpoint = iter.Next()
			Ω(foundEndpoint).ToNot(BeNil())
			Ω(foundEndpoint).To(Equal(endpointBar))
		})

		It("returns the next available endpoint when the initial is not found", func() {
			eFoo := NewEndpoint("", "1.2.3.4", 1234, "foo", nil)
			pool.Put(eFoo)

			iter := pool.Endpoints("bogus")
			e := iter.Next()
			Ω(e).ShouldNot(BeNil())
			Ω(e).Should(Equal(eFoo))
		})

		It("finds the correct endpoint when private ids change", func() {
			endpointFoo := NewEndpoint("", "1.2.3.4", 1234, "foo", nil)
			pool.Put(endpointFoo)

			iter := pool.Endpoints(endpointFoo.PrivateInstanceId)
			foundEndpoint := iter.Next()
			Ω(foundEndpoint).ShouldNot(BeNil())
			Ω(foundEndpoint).Should(Equal(endpointFoo))

			endpointBar := NewEndpoint("", "1.2.3.4", 1234, "bar", nil)
			pool.Put(endpointBar)

			iter = pool.Endpoints("foo")
			foundEndpoint = iter.Next()
			Ω(foundEndpoint).ShouldNot(Equal(endpointFoo))

			iter = pool.Endpoints("bar")
			Ω(foundEndpoint).Should(Equal(endpointBar))
		})
	})

	Describe("Failed", func() {
		It("skips failed endpoints", func() {
			e1 := NewEndpoint("", "1.2.3.4", 5678, "", nil)
			e2 := NewEndpoint("", "5.6.7.8", 1234, "", nil)
			pool.Put(e1)
			pool.Put(e2)

			iter := pool.Endpoints("")
			n := iter.Next()
			Ω(n).ShouldNot(BeNil())

			iter.EndpointFailed()

			nn1 := iter.Next()
			nn2 := iter.Next()
			Ω(nn1).ShouldNot(BeNil())
			Ω(nn2).ShouldNot(BeNil())
			Ω(nn1).ShouldNot(Equal(n))
			Ω(nn1).Should(Equal(nn2))
		})

		It("resets when all endpoints are failed", func() {
			e1 := NewEndpoint("", "1.2.3.4", 5678, "", nil)
			e2 := NewEndpoint("", "5.6.7.8", 1234, "", nil)
			pool.Put(e1)
			pool.Put(e2)

			iter := pool.Endpoints("")
			n1 := iter.Next()
			iter.EndpointFailed()
			n2 := iter.Next()
			iter.EndpointFailed()
			Ω(n1).ShouldNot(Equal(n2))

			n1 = iter.Next()
			n2 = iter.Next()
			Ω(n1).ShouldNot(Equal(n2))
		})

		It("resets failed endpoints after exceeding failure duration", func() {
			pool = NewPool(50 * time.Millisecond)

			e1 := NewEndpoint("", "1.2.3.4", 5678, "", nil)
			e2 := NewEndpoint("", "5.6.7.8", 1234, "", nil)
			pool.Put(e1)
			pool.Put(e2)

			iter := pool.Endpoints("")
			n1 := iter.Next()
			n2 := iter.Next()
			Ω(n1).ShouldNot(Equal(n2))

			iter.EndpointFailed()

			n1 = iter.Next()
			n2 = iter.Next()
			Ω(n1).Should(Equal(n2))

			time.Sleep(50 * time.Millisecond)

			n1 = iter.Next()
			n2 = iter.Next()
			Ω(n1).ShouldNot(Equal(n2))
		})
	})
})
