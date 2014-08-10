package common_test

import (
	. "github.com/cloudfoundry/gorouter/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"encoding/json"
	"fmt"
	"time"
)

var _ = Describe("Durations", func() {
	Context("Duration", func() {
		It("supports JSON", func() {
			d := Duration(123456)
			var i interface{} = &d

			_, ok := i.(json.Marshaler)
			Ω(ok).Should(BeTrue())

			_, ok = i.(json.Unmarshaler)
			Ω(ok).Should(BeTrue())
		})

		It("marshals JSON", func() {
			d := Duration(time.Hour*36 + time.Second*10)
			b, err := json.Marshal(d)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(b)).To(Equal(`"1d:12h:0m:10s"`))
		})

		It("unmarshals JSON", func() {
			d := Duration(time.Hour*36 + time.Second*20)
			b, err := json.Marshal(d)
			Ω(err).ShouldNot(HaveOccurred())

			var dd Duration
			dd.UnmarshalJSON(b)
			Ω(dd).To(Equal(d))
		})
	})

	Context("Time", func() {
		It("marshals JSON", func() {
			n := time.Now()
			f := "2006-01-02 15:04:05 -0700"

			t := Time(n)
			b, e := json.Marshal(t)
			Ω(e).ShouldNot(HaveOccurred())
			Ω(string(b)).To(Equal(fmt.Sprintf(`"%s"`, n.Format(f))))
		})

		It("unmarshals JSON", func() {
			t := Time(time.Unix(time.Now().Unix(), 0)) // The precision of Time is 'second'
			b, err := json.Marshal(t)
			Ω(err).ShouldNot(HaveOccurred())

			var tt Time
			err = tt.UnmarshalJSON(b)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(tt).To(Equal(t))
		})
	})
})
