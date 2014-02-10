package drain_test

import (
	"time"

	. "github.com/onsi/ginkgo"

	"github.com/pivotal-cf-experimental/garden/drain"
)

var _ = Describe("Drain", func() {
	Describe(".Wait", func() {
		It("returns immediately", func(done Done) {
			drain := drain.New()
			drain.Wait()
			close(done)
		}, 1.0)
	})

	Describe(".Incr", func() {
		Describe(".Wait", func() {
			It("blocks", func() {
				drain := drain.New()
				drain.Incr()

				waited := make(chan bool)

				go func() {
					drain.Wait()
					waited <- true
				}()

				select {
				case <-waited:
					Fail("did not wait!")
				case <-time.After(100 * time.Millisecond):
				}
			})

			Describe(".Wait", func() {
				It("blocks", func() {
					drain := drain.New()
					drain.Incr()

					waited := make(chan bool)

					go func() {
						drain.Wait()
						waited <- true
					}()

					go func() {
						drain.Wait()
						waited <- true
					}()

					select {
					case <-waited:
						Fail("did not wait!")
					case <-time.After(100 * time.Millisecond):
					}
				})

				Describe(".Decr", func() {
					It("causes both .Waits to return", func(done Done) {
						drain := drain.New()
						drain.Incr()

						waited := make(chan bool)

						go func() {
							drain.Wait()
							waited <- true
						}()

						go func() {
							drain.Wait()
							waited <- true
						}()

						drain.Decr()

						select {
						case <-waited:
							select {
							case <-waited:
								close(done)

							case <-time.After(100 * time.Millisecond):
								Fail("wait blocked!")
							}
						case <-time.After(100 * time.Millisecond):
							Fail("wait blocked!")
						}
					}, 1.0)
				})
			})

			Describe(".Decr", func() {
				It("causes .Wait to return", func(done Done) {
					drain := drain.New()
					drain.Incr()

					waited := make(chan bool)

					go func() {
						drain.Wait()
						waited <- true
					}()

					drain.Decr()

					select {
					case <-waited:
						close(done)
					case <-time.After(100 * time.Millisecond):
						Fail("wait blocked!")
					}
				}, 1.0)
			})
		})

		Describe(".Decr", func() {
			Describe(".Wait", func() {
				It("returns immediately", func(done Done) {
					drain := drain.New()
					drain.Incr()
					drain.Decr()
					drain.Wait()
					close(done)
				}, 1.0)
			})
		})

		Describe(".Incr", func() {
			Describe(".Decr", func() {
				Describe(".Wait", func() {
					It("blocks", func() {
						drain := drain.New()
						drain.Incr()
						drain.Incr()
						drain.Decr()

						waited := make(chan bool)

						go func() {
							drain.Wait()
							waited <- true
						}()

						select {
						case <-waited:
							Fail("did not wait!")
						case <-time.After(100 * time.Millisecond):
						}
					})
				})
			})
		})
	})
})
