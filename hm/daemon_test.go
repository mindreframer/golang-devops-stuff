package hm_test

import (
	"errors"
	"time"
	. "github.com/cloudfoundry/hm9000/hm"
	"github.com/cloudfoundry/hm9000/testhelpers/fakelogger"
	"github.com/cloudfoundry/storeadapter/fakestoreadapter"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Daemon", func() {
	var adapter *fakestoreadapter.FakeStoreAdapter

	BeforeEach(func() {
		adapter = fakestoreadapter.New()
	})

	It("should call the function every PERIOD seconds, unless the function takes *longer* than PERIOD, and it should timeout when the function takes *too* long", func(done Done) {
		defer close(done)

		go func() {
			released := <-adapter.ReleaseNodeChannel
			released <- true
		}()

		adapter.MaintainNodeStatus <- true

		callTimes := []float64{}
		startTime := time.Now()
		i := 0
		err := Daemonize("Daemon Test", func() error {
			callTimes = append(callTimes, time.Since(startTime).Seconds())
			i += 1
			time.Sleep(time.Duration(i*10) * time.Millisecond)
			return nil
		}, 20*time.Millisecond, 35*time.Millisecond, fakelogger.NewFakeLogger(), adapter)

		Ω(callTimes).Should(HaveLen(4))

		Ω(callTimes[0]).Should(BeNumerically("~", 0, 0.01), "The first call happens immediately and sleeps for 10 seconds")
		Ω(callTimes[1]).Should(BeNumerically("~", 0.02, 0.01), "The second call happens after PERIOD and sleeps for 20 seconds")
		Ω(callTimes[2]).Should(BeNumerically("~", 0.04, 0.01), "The third call happens after PERIOD and sleeps for 30 seconds")
		Ω(callTimes[3]).Should(BeNumerically("~", 0.07, 0.01), "The fourth call waits for function to finish and happens after 30 seconds (> PERIOD) and sleeps for 40 seconds which...")
		Ω(err).Should(Equal(errors.New("Daemon timed out. Aborting!")), "..causes a timeout")
	})

	It("acquires the lock once", func() {
		go Daemonize(
			"ComponentName",
			func() error { return nil },
			20*time.Millisecond,
			35*time.Millisecond,
			fakelogger.NewFakeLogger(),
			adapter,
		)

		Eventually(func() string { return adapter.MaintainedNodeName }).Should(Equal("/hm/locks/ComponentName"))
	})

	Context("when the locker fails", func() {
		disaster := errors.New("oh no!")

		BeforeEach(func() {
			adapter.MaintainNodeError = disaster
		})

		It("returns the error", func() {
			err := Daemonize(
				"Daemon Test",
				func() error { Fail("NOPE"); return nil },
				20*time.Millisecond,
				35*time.Millisecond,
				fakelogger.NewFakeLogger(),
				adapter,
			)

			Ω(err).Should(Equal(disaster))
		})
	})

	Context("when the callback times out", func() {
		It("releases the lock", func(done Done) {
			defer close(done)

			didRelease := make(chan bool)

			go func() {
				released := <-adapter.ReleaseNodeChannel
				released <- true
				didRelease <- true
			}()

			adapter.MaintainNodeStatus <- true

			Daemonize(
				"Daemon Test",
				func() error { time.Sleep(1 * time.Second); return nil },
				20*time.Millisecond,
				35*time.Millisecond,
				fakelogger.NewFakeLogger(),
				adapter,
			)

			<-didRelease
		})
	})
})
