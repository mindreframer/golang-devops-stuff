package http_test

import (
	. "github.com/cloudfoundry/gorouter/common/http"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"net"
	"net/http"
)

var _ = Describe("http", func() {
	var listener net.Listener

	AfterEach(func() {
		if listener != nil {
			listener.Close()
		}
	})

	bootstrap := func(x Authenticator) *http.Request {
		var err error

		h := func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}

		y := &BasicAuth{http.HandlerFunc(h), x}

		z := &http.Server{Handler: y}

		l, err := net.Listen("tcp", "127.0.0.1:0")
		Ω(err).ShouldNot(HaveOccurred())

		go z.Serve(l)

		// Keep listener around such that test teardown can close it
		listener = l

		r, err := http.NewRequest("GET", "http://"+l.Addr().String(), nil)
		Ω(err).ShouldNot(HaveOccurred())
		return r
	}

	Context("Unauthorized", func() {
		It("without credentials", func() {
			req := bootstrap(nil)

			resp, err := http.DefaultClient.Do(req)
			Ω(err).ShouldNot(HaveOccurred())

			Ω(resp.StatusCode).Should(Equal(http.StatusUnauthorized))
		})

		It("with invalid header", func() {
			req := bootstrap(nil)

			req.Header.Set("Authorization", "invalid")

			resp, err := http.DefaultClient.Do(req)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(resp.StatusCode).Should(Equal(http.StatusUnauthorized))
		})

		It("with bad credentials", func() {
			f := func(u, p string) bool {
				Ω(u).Should(Equal("user"))
				Ω(p).Should(Equal("bad"))
				return false
			}

			req := bootstrap(f)

			req.SetBasicAuth("user", "bad")

			resp, err := http.DefaultClient.Do(req)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(resp.StatusCode).Should(Equal(http.StatusUnauthorized))
		})
	})
	It("succeeds with good credentials", func() {
		f := func(u, p string) bool {
			Ω(u).Should(Equal("user"))
			Ω(p).Should(Equal("good"))
			return true
		}

		req := bootstrap(f)

		req.SetBasicAuth("user", "good")

		resp, err := http.DefaultClient.Do(req)
		Ω(err).ShouldNot(HaveOccurred())
		Ω(resp.StatusCode).Should(Equal(http.StatusOK))
	})
})
