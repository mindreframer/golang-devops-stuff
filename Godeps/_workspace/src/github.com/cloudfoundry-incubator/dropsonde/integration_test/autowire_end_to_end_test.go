package integration_test

import (
	"code.google.com/p/gogoprotobuf/proto"
	"fmt"
	"github.com/cloudfoundry-incubator/dropsonde/autowire"
	"github.com/cloudfoundry-incubator/dropsonde/events"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"net"
	"net/http"
	"os"
	"sync"
)

// these tests need to be invoked individually from an external script,
// since environment variables need to be set/unset before starting the tests
var _ = Describe("Autowire End-to-End", func() {
	Context("with DROPSONDE_ORIGIN set", func() {
		var oldEnv string

		BeforeEach(func() {
			oldEnv = os.Getenv("DROPSONDE_ORIGIN")
			os.Setenv("DROPSONDE_ORIGIN", "test-origin")
			autowire.Initialize()
		})

		AfterEach(func() {
			os.Setenv("DROPSONDE_ORIGIN", oldEnv)
		})

		It("emits HTTP client/server events and heartbeats", func() {
			udpListener, err := net.ListenPacket("udp4", ":42420")
			Expect(err).ToNot(HaveOccurred())
			defer udpListener.Close()
			udpDataChan := make(chan []byte, 16)

			receivedEvents := make(map[string]bool)
			lock := sync.RWMutex{}
			origin := os.Getenv("DROPSONDE_ORIGIN")

			go func() {
				defer close(udpDataChan)
				for {
					buffer := make([]byte, 1024)
					n, _, err := udpListener.ReadFrom(buffer)
					if err != nil {
						return
					}

					if n == 0 {
						panic("Received empty packet")
					}
					envelope := new(events.Envelope)
					err = proto.Unmarshal(buffer[0:n], envelope)
					if err != nil {
						panic(err)
					}

					var eventId = envelope.GetEventType().String()

					switch envelope.GetEventType() {
					case events.Envelope_HttpStart:
						eventId += envelope.GetHttpStart().GetPeerType().String()
					case events.Envelope_HttpStop:
						eventId += envelope.GetHttpStop().GetPeerType().String()
					case events.Envelope_Heartbeat:
					default:
						panic("Unexpected message type")

					}

					if envelope.GetOrigin() != origin {
						panic("origin not as expected")
					}

					func() {
						lock.Lock()
						defer lock.Unlock()
						receivedEvents[eventId] = true
					}()
				}
			}()

			httpListener, err := net.Listen("tcp", "localhost:0")
			Expect(err).ToNot(HaveOccurred())
			defer httpListener.Close()
			httpHandler := autowire.InstrumentedHandler(FakeHandler{})
			go http.Serve(httpListener, httpHandler)

			_, err = http.Get("http://" + httpListener.Addr().String())
			Expect(err).ToNot(HaveOccurred())

			expectedEventTypes := []string{"HttpStartClient", "HttpStartServer", "HttpStopServer", "HttpStopClient"}

			for _, eventType := range expectedEventTypes {
				Eventually(func() bool {
					lock.RLock()
					defer lock.RUnlock()
					_, ok := receivedEvents[eventType]
					return ok
				}).Should(BeTrue())
			}

			Eventually(func() bool {
				lock.RLock()
				defer lock.RUnlock()
				_, ok := receivedEvents["Heartbeat"]
				return ok
			}).Should(BeTrue())
		})
	})
})

type FakeHandler struct{}

func (fh FakeHandler) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(rw, "Hello")
}

type FakeRoundTripper struct{}

func (frt FakeRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	return nil, nil
}
