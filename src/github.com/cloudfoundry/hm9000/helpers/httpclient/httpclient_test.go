package httpclient_test

import (
	"fmt"
	. "github.com/cloudfoundry/hm9000/helpers/httpclient"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"io/ioutil"
	"net"
	"net/http"
	"strconv"
	"time"
)

func init() {
	net.Listen("tcp", ":8887")

	http.HandleFunc("/sleep", func(w http.ResponseWriter, r *http.Request) {
		sleepTimeInSeconds, _ := strconv.ParseFloat(r.URL.Query().Get("time"), 64)
		time.Sleep(time.Duration(sleepTimeInSeconds * float64(time.Second)))
		fmt.Fprintf(w, "I'm awake!")
	})

	go http.ListenAndServe(":8889", nil)
}

var _ = Describe("Httpclient", func() {
	var client HttpClient

	BeforeEach(func() {
		client = NewHttpClient(1 * time.Millisecond)
	})

	Context("when the request times out (trying to connect)", func() {
		It("should return an appropriate timeout error", func(done Done) {
			request, _ := http.NewRequest("GET", "http://127.0.0.1:8887/", nil)
			client.Do(request, func(response *http.Response, err error) {
				Ω(err).Should(HaveOccurred())
				close(done)
			})
		}, 0.1)
	})

	Context("when the request times out (after conecting)", func() {
		It("should return an appropriate timeout error", func() {
			request, _ := http.NewRequest("GET", "http://127.0.0.1:8889/sleep?time=1", nil)
			client.Do(request, func(response *http.Response, err error) {
				Ω(err).Should(HaveOccurred())
			})
		})
	})

	Context("when the request does not time out", func() {
		It("should return the correct response", func() {
			request, _ := http.NewRequest("GET", "http://127.0.0.1:8889/sleep?time=0", nil)
			client.Do(request, func(response *http.Response, err error) {
				Ω(err).ShouldNot(HaveOccurred())
				defer response.Body.Close()
				body, err := ioutil.ReadAll(response.Body)
				Ω(err).ShouldNot(HaveOccurred())
				Ω(string(body)).Should(Equal("I'm awake!"))
			})
		})
	})
})
