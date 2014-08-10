package sigar_test

import (
	"io/ioutil"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	sigar "github.com/cloudfoundry/gosigar"
)

var _ = Describe("sigarLinux", func() {
	Describe("CPU", func() {
		var (
			statFile string
			cpu      sigar.Cpu
		)

		BeforeEach(func() {
			procd, err := ioutil.TempDir("", "sigarTests")
			Expect(err).ToNot(HaveOccurred())
			sigar.Procd = procd
			statFile = procd + "/stat"

			cpu = sigar.Cpu{}

			statContents := []byte("cpu 25 1 2 3 4 5 6 7")
			err = ioutil.WriteFile(statFile, statContents, 0644)
			Expect(err).ToNot(HaveOccurred())
		})

		AfterEach(func() {
			sigar.Procd = "/proc"
		})

		Describe("Get", func() {
			It("gets CPU usage", func() {
				err := cpu.Get()
				Expect(err).ToNot(HaveOccurred())
				Expect(cpu.User).To(Equal(uint64(25)))
			})
		})

		Describe("CollectCpuStats", func() {
			It("collects CPU usage over time", func() {
				concreteSigar := &sigar.ConcreteSigar{}
				cpuUsages, stop := concreteSigar.CollectCpuStats(500 * time.Millisecond)

				Expect(<-cpuUsages).To(Equal(sigar.Cpu{
					User:    uint64(25),
					Nice:    uint64(1),
					Sys:     uint64(2),
					Idle:    uint64(3),
					Wait:    uint64(4),
					Irq:     uint64(5),
					SoftIrq: uint64(6),
					Stolen:  uint64(7),
				}))

				statContents := []byte("cpu 30 3 7 10 25 55 36 65")
				err := ioutil.WriteFile(statFile, statContents, 0644)
				Expect(err).ToNot(HaveOccurred())

				Expect(<-cpuUsages).To(Equal(sigar.Cpu{
					User:    uint64(5),
					Nice:    uint64(2),
					Sys:     uint64(5),
					Idle:    uint64(7),
					Wait:    uint64(21),
					Irq:     uint64(50),
					SoftIrq: uint64(30),
					Stolen:  uint64(58),
				}))

				stop <- struct{}{}
			})
		})
	})
})
