package udp_listener_test

import (
	"fmt"
	"github.com/cloudfoundry-incubator/dropsonde/udp_listener"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net"
)

var _ = Describe("UdpListener", func() {
	Describe("Run", func() {
		var (
			stopChan chan struct{}
			errChan  chan error
			dataChan chan []byte
		)

		BeforeEach(func() {
			stopChan = make(chan struct{})
			errChan = make(chan error)
			dataChan = make(chan []byte)

			udp_listener.UdpListeningPort.Set(0)
		})

		Context("when listening works", func() {
			var conn *net.UDPConn

			JustBeforeEach(func() {
				go func() {
					defer close(errChan)
					err := udp_listener.Run(dataChan, stopChan)
					errChan <- err
				}()

				Eventually(udp_listener.UdpListeningPort.Get).ShouldNot(BeZero())

				addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("localhost:%d", udp_listener.UdpListeningPort.Get()))
				Expect(err).ToNot(HaveOccurred())

				conn, err = net.DialUDP("udp", nil, addr)
				Expect(err).ToNot(HaveOccurred())
			})

			AfterEach(func() {
				Eventually(errChan).Should(BeClosed())
			})

			It("listens for UDP packets and puts them on the data channel", func(done Done) {
				defer close(done)

				sentData := []byte("test-data")

				conn.Write(sentData)

				receivedData := <-dataChan
				Expect(receivedData).To(Equal(sentData))
				close(stopChan)
				err := <-errChan
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("when listening fails", func() {
			Context("udp", func() {
				BeforeEach(func() {
					netConn, err := net.ListenPacket("udp", ":0")
					Expect(err).ToNot(HaveOccurred())
					port := netConn.LocalAddr().(*net.UDPAddr).Port
					udp_listener.UdpListeningPort.Set(port)
				})

				It("returns an error", func(done Done) {
					defer close(done)
					var err error
					runDoneChan := make(chan struct{})
					go func() {
						err = udp_listener.Run(dataChan, stopChan)
						close(runDoneChan)
					}()
					<-runDoneChan
					Expect(err).To(HaveOccurred())
				})
			})
		})
	})
})
