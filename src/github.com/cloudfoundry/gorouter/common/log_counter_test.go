package common_test

import (
	"encoding/json"
	. "github.com/cloudfoundry/gorouter/common"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	steno "github.com/cloudfoundry/gosteno"
)

var _ = Describe("LogCounter", func() {
	var info = steno.NewRecord("", steno.LOG_INFO, "", nil)
	var err = steno.NewRecord("", steno.LOG_ERROR, "", nil)

	It("counts the number of records", func() {
		counter := NewLogCounter()
		counter.AddRecord(info)
		Ω(counter.GetCount(steno.LOG_INFO.Name)).To(Equal(1))

		counter.AddRecord(info)
		Ω(counter.GetCount(steno.LOG_INFO.Name)).To(Equal(2))
	})

	It("counts all log levels", func() {
		counter := NewLogCounter()
		counter.AddRecord(info)
		Ω(counter.GetCount(steno.LOG_INFO.Name)).To(Equal(1))

		counter.AddRecord(err)
		Ω(counter.GetCount(steno.LOG_ERROR.Name)).To(Equal(1))
	})

	It("marshals the set of counts", func() {
		counter := NewLogCounter()
		counter.AddRecord(info)
		counter.AddRecord(err)

		b, e := counter.MarshalJSON()
		Ω(e).ShouldNot(HaveOccurred())

		v := map[string]int{}
		e = json.Unmarshal(b, &v)
		Ω(e).ShouldNot(HaveOccurred())
		Ω(v).Should(HaveLen(2))
		Ω(v[steno.LOG_INFO.Name]).Should(Equal(1))
		Ω(v[steno.LOG_ERROR.Name]).Should(Equal(1))
	})
})
