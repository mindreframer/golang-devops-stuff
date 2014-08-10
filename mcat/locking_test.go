package mcat_test

import (
	"time"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gbytes"
	"github.com/onsi/gomega/gexec"
)

var _ = Describe("Locking", func() {
	Describe("vieing for the lock", func() {
		Context("when two long-lived processes try to run", func() {
			It("one waits for the other to exit and then grabs the lock", func() {
				listenerA := cliRunner.StartSession("listen", 1)

				Eventually(listenerA, 10*time.Second).Should(gbytes.Say("Acquired lock"))

				defer func() {
					listenerA.Interrupt().Wait()
				}()

				listenerB := cliRunner.StartSession("listen", 1)
				defer func() {
					listenerB.Interrupt().Wait()
				}()

				Eventually(listenerB, 10*time.Second).Should(gbytes.Say("Acquiring"))
				Consistently(listenerB).ShouldNot(gbytes.Say("Acquired"))

				listenerA.Interrupt().Wait()

				coordinator.StoreRunner.FastForwardTime(10)

				Eventually(listenerB, 20*time.Second).Should(gbytes.Say("Acquired"))
			})
		})

		Context("when two polling processes try to run", func() {
			It("one waits for the other to exit and then grabs the lock", func() {
				analyzerA := cliRunner.StartSession("analyze", 1, "--poll")
				defer func() {
					analyzerA.Interrupt().Wait()
				}()

				Eventually(analyzerA, 10*time.Second).Should(gbytes.Say("Acquired lock"))

				analyzerB := cliRunner.StartSession("analyze", 1, "--poll")
				defer func() {
					analyzerB.Interrupt().Wait()
				}()

				Eventually(analyzerB, 10*time.Second).Should(gbytes.Say("Acquiring"))
				Consistently(analyzerB).ShouldNot(gbytes.Say("Acquired"))

				analyzerA.Interrupt().Wait()

				coordinator.StoreRunner.FastForwardTime(10)

				Eventually(analyzerB, 20*time.Second).Should(gbytes.Say("Acquired"))
			})
		})
	})

	Context("when the lock disappears", func() {
		Context("long-lived processes", func() {
			It("should exit 197", func() {
				listenerA := cliRunner.StartSession("listen", 1)
				defer func() {
					listenerA.Interrupt().Wait()
				}()

				Eventually(listenerA, 10*time.Second).Should(gbytes.Say("Acquired lock"))

				coordinator.StoreAdapter.Delete("/hm/locks")

				Eventually(listenerA, 10*time.Second).Should(gbytes.Say("Lost the lock"))
				Eventually(listenerA, 20*time.Second).Should(gexec.Exit(197))
			})
		})

		Context("polling processes", func() {
			It("should exit 197", func() {
				analyzerA := cliRunner.StartSession("analyze", 1, "--poll")
				defer func() {
					analyzerA.Interrupt().Wait()
				}()

				Eventually(analyzerA, 10*time.Second).Should(gbytes.Say("Acquired lock"))

				coordinator.StoreAdapter.Delete("/hm/locks")

				Eventually(analyzerA, 10*time.Second).Should(gbytes.Say("Lost the lock"))
				Eventually(analyzerA, 20*time.Second).Should(gexec.Exit(197))
			})
		})
	})
})
