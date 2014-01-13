package measurements_test

import (
	"fmt"
	"net"
	"os"
	"runtime"
	"sync/atomic"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"github.com/vito/gordon"
	"github.com/vito/gordon/warden"
)

var _ = Describe("The Warden server", func() {
	var wardenClient *gordon.Client

	runtime.GOMAXPROCS(runtime.NumCPU())

	BeforeEach(func() {
		socketPath := os.Getenv("WARDEN_TEST_SOCKET")
		Eventually(ErrorDialingUnix(socketPath)).ShouldNot(HaveOccurred())

		wardenClient = gordon.NewClient(&gordon.ConnectionInfo{SocketPath: socketPath})

		err := wardenClient.Connect()
		Expect(err).ToNot(HaveOccurred())
	})

	Describe("streaming output from a chatty job", func() {
		var handle string

		BeforeEach(func() {
			res, err := wardenClient.Create()
			Expect(err).ToNot(HaveOccurred())

			handle = res.GetHandle()
		})

		streamCounts := []int{0}

		for i := 1; i <= 128; i *= 2 {
			streamCounts = append(streamCounts, i)
		}

		for _, streams := range streamCounts {
			Context(fmt.Sprintf("with %d streams", streams), func() {
				var started time.Time
				var receivedBytes uint64

				numToSpawn := streams

				BeforeEach(func() {
					receivedBytes = 0
					started = time.Now()

					spawned := make(chan bool)

					for j := 0; j < numToSpawn; j++ {
						go func() {
							spawnRes, err := wardenClient.Spawn(
								handle,
								"cat /dev/zero",
								true,
							)
							Expect(err).ToNot(HaveOccurred())

							results, err := wardenClient.Stream(handle, spawnRes.GetJobId())
							Expect(err).ToNot(HaveOccurred())

							go func(results chan *warden.StreamResponse) {
								for {
									res, ok := <-results
									if !ok {
										break
									}

									atomic.AddUint64(&receivedBytes, uint64(len(res.GetData())))
								}
							}(results)

							spawned <- true
						}()
					}

					for j := 0; j < numToSpawn; j++ {
						<-spawned
					}
				})

				AfterEach(func() {
					_, err := wardenClient.Destroy(handle)
					Expect(err).ToNot(HaveOccurred())
				})

				Measure("it should not adversely affect the rest of the API", func(b Benchmarker) {
					var newHandle string

					b.Time("creating another container", func() {
						res, err := wardenClient.Create()
						Expect(err).ToNot(HaveOccurred())

						newHandle = res.GetHandle()
					})

					for i := 0; i < 10; i++ {
						b.Time("getting container info (10x)", func() {
							_, err := wardenClient.Info(newHandle)
							Expect(err).ToNot(HaveOccurred())
						})
					}

					for i := 0; i < 10; i++ {
						b.Time("running a job (10x)", func() {
							_, err := wardenClient.Run(newHandle, "ls")
							Expect(err).ToNot(HaveOccurred())
						})
					}

					b.Time("destroying the container", func() {
						_, err := wardenClient.Destroy(newHandle)
						Expect(err).ToNot(HaveOccurred())
					})

					b.RecordValue(
						"received rate (bytes/second)",
						float64(receivedBytes)/float64(time.Since(started)/time.Second),
					)

					fmt.Println("total time:", time.Since(started))
				}, 5)
			})
		}
	})
})

func ErrorDialingUnix(socketPath string) func() error {
	return func() error {
		conn, err := net.Dial("unix", socketPath)
		if err == nil {
			conn.Close()
		}

		return err
	}
}
