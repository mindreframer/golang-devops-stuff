package timebomb_test

import (
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/pivotal-cf-experimental/garden/server/timebomb"
)

var _ = Describe("THE TIMEBOMB", func() {
	Context("WHEN STRAPPED", func() {
		It("DETONATES AFTER THE COUNTDOWN", func() {
			detonated := make(chan time.Time)

			countdown := 100 * time.Millisecond

			bomb := timebomb.New(
				countdown,
				func() {
					detonated <- time.Now()
				},
			)

			before := time.Now()

			bomb.Strap()

			Expect((<-detonated).Sub(before)).To(BeNumerically(">=", countdown))
		})

		It("DOES NOT DETONATE AGAIN", func() {
			detonated := make(chan time.Time)

			countdown := 100 * time.Millisecond

			bomb := timebomb.New(
				countdown,
				func() {
					detonated <- time.Now()
				},
			)

			before := time.Now()

			bomb.Strap()

			Expect((<-detonated).Sub(before)).To(BeNumerically(">=", countdown))

			delay := 50 * time.Millisecond

			select {
			case <-detonated:
				Fail("MILLIONS ARE DEAD...AGAIN")
			case <-time.After(countdown + delay):
			}
		})

		Context("AND THEN DEFUSED", func() {
			It("DOES NOT DETONATE", func() {
				detonated := make(chan time.Time)

				countdown := 100 * time.Millisecond

				bomb := timebomb.New(
					countdown,
					func() {
						detonated <- time.Now()
					},
				)

				bomb.Strap()
				bomb.Defuse()

				delay := 50 * time.Millisecond

				select {
				case <-detonated:
					Fail("MILLIONS ARE DEAD")
				case <-time.After(countdown + delay):
				}
			})
		})

		Context("AND THEN PAUSED", func() {
			It("DOES NOT DETONATE", func() {
				detonated := make(chan time.Time)

				countdown := 100 * time.Millisecond

				bomb := timebomb.New(
					countdown,
					func() {
						detonated <- time.Now()
					},
				)

				bomb.Strap()
				bomb.Pause()

				delay := 50 * time.Millisecond

				select {
				case <-detonated:
					Fail("MILLIONS ARE DEAD")
				case <-time.After(countdown + delay):
				}
			})

			Context("AND THEN UNPAUSED", func() {
				It("DETONATES AFTER THE COUNTDOWN", func() {
					detonated := make(chan time.Time)

					countdown := 100 * time.Millisecond

					bomb := timebomb.New(
						countdown,
						func() {
							detonated <- time.Now()
						},
					)

					before := time.Now()

					bomb.Strap()

					bomb.Pause()

					delay := 50 * time.Millisecond

					time.Sleep(delay)

					bomb.Unpause()

					Expect((<-detonated).Sub(before)).To(BeNumerically(">=", countdown+delay))
				})

				Context("AND THEN PAUSED AGAIN", func() {
					It("DOES NOT DETONATE", func() {
						detonated := make(chan time.Time)

						countdown := 100 * time.Millisecond

						bomb := timebomb.New(
							countdown,
							func() {
								detonated <- time.Now()
							},
						)

						bomb.Strap()
						bomb.Pause()
						bomb.Unpause()
						bomb.Pause()

						delay := 50 * time.Millisecond

						select {
						case <-detonated:
							Fail("MILLIONS ARE DEAD")
						case <-time.After(countdown + delay):
						}
					})
				})
			})

			Context("TWICE", func() {
				Context("AND THEN UNPAUSED", func() {
					It("DOES NOT DETONATE", func() {
						detonated := make(chan time.Time)

						countdown := 100 * time.Millisecond

						bomb := timebomb.New(
							countdown,
							func() {
								detonated <- time.Now()
							},
						)

						bomb.Strap()
						bomb.Pause()
						bomb.Pause()
						bomb.Unpause()

						delay := 50 * time.Millisecond

						select {
						case <-detonated:
							Fail("MILLIONS ARE DEAD")
						case <-time.After(countdown + delay):
						}
					})

					Context("TWICE", func() {
						It("DETONATES AFTER THE COUNTDOWN", func() {
							detonated := make(chan time.Time)

							countdown := 100 * time.Millisecond

							bomb := timebomb.New(
								countdown,
								func() {
									detonated <- time.Now()
								},
							)

							before := time.Now()

							bomb.Strap()

							bomb.Pause()
							bomb.Pause()

							bomb.Unpause()

							delay := 50 * time.Millisecond

							time.Sleep(delay)

							bomb.Unpause()

							Expect((<-detonated).Sub(before)).To(BeNumerically(">=", countdown+delay))
						})
					})
				})
			})
		})
	})
})
