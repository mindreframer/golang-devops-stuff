package workerpool_test

import (
	. "github.com/cloudfoundry/hm9000/helpers/workerpool"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"runtime"
	"time"
)

var _ = Describe("WorkerPool", func() {
	var pool *WorkerPool

	BeforeEach(func() {
		poolSize := 2
		pool = NewWorkerPool(poolSize)
	})

	Describe("scheduling work", func() {
		Context("when passed one function", func() {
			It("should run the passed in function", func() {
				called := make(chan bool, 1)

				pool.ScheduleWork(func() {
					called <- true
				})

				Eventually(called, 0.1, 0.01).Should(HaveLen(1))
			})
		})

		Context("when passed many function", func() {
			var (
				startTime time.Time
				runTimes  chan time.Duration
				sleepTime time.Duration
				work      func()
			)

			BeforeEach(func() {
				startTime = time.Now()
				runTimes = make(chan time.Duration, 10)
				sleepTime = time.Duration(0.01 * float64(time.Second))

				work = func() {
					time.Sleep(sleepTime)
					runTimes <- time.Since(startTime)
				}
			})

			Context("when passed poolSize functions", func() {
				BeforeEach(func() {
					pool.ScheduleWork(work)
					pool.ScheduleWork(work)
				})

				It("should run the functions concurrently", func() {
					Eventually(runTimes, 0.1, 0.01).Should(HaveLen(2))
					Ω(<-runTimes).Should(BeNumerically("<=", sleepTime+sleepTime/2))
					Ω(<-runTimes).Should(BeNumerically("<=", sleepTime+sleepTime/2))
				})
			})

			Context("when passed more than poolSize functions", func() {
				BeforeEach(func() {
					pool.ScheduleWork(work)
					pool.ScheduleWork(work)
					pool.ScheduleWork(work)
				})

				It("should run all the functions, but at most poolSize at a time", func() {
					Eventually(runTimes, 0.1, 0.01).Should(HaveLen(3))

					//first batch
					Ω(<-runTimes).Should(BeNumerically("<=", sleepTime+sleepTime/2))
					Ω(<-runTimes).Should(BeNumerically("<=", sleepTime+sleepTime/2))

					//second batch
					Ω(<-runTimes).Should(BeNumerically(">=", sleepTime*2))
				})
			})
		})

		Context("when stopped", func() {
			var numGoroutines int

			BeforeEach(func() {
				numGoroutines = runtime.NumGoroutine()
				pool.StopWorkers()
			})

			It("should never perform the work", func() {
				called := make(chan bool, 1)

				pool.ScheduleWork(func() {
					called <- true
				})

				time.Sleep(time.Duration(0.1 * float64(time.Second)))

				Ω(called).Should(HaveLen(0))
			})

			It("should stop the workers", func() {
				Eventually(runtime.NumGoroutine, 0.1, 0.01).Should(Equal(numGoroutines-2), "Should have reduced number of go routines by pool size")
			})
		})
	})

	Describe("Measuring capacity", func() {
		It("should track the ...", func() {
			dt := time.Duration(0.1 * float64(time.Second))

			pool.StartTrackingUsage()
			called := make(chan bool, 1)
			pool.ScheduleWork(func() {
				time.Sleep(dt)
				called <- true
			})

			<-called
			time.Sleep(dt)

			usage, duration := pool.MeasureUsage()

			Ω(duration.Seconds()).Should(BeNumerically("~", 0.2, 0.05))
			Ω(usage).Should(BeNumerically("~", 0.25, 0.05)) //one of two workers was occupied for half the time:

			//now do it again, but occupy two workers

			called = make(chan bool, 1)
			pool.ScheduleWork(func() {
				time.Sleep(dt)
				called <- true
			})

			pool.ScheduleWork(func() {
				time.Sleep(dt)
				called <- true
			})

			<-called
			time.Sleep(dt)

			usage, duration = pool.MeasureUsage()
			Ω(duration.Seconds()).Should(BeNumerically("~", 0.2, 0.05))
			Ω(usage).Should(BeNumerically("~", 0.5, 0.05)) //both workers were occupied for half the time:
		})
	})
})
