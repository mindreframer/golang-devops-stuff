package loggregatorclient

import (
	"github.com/cloudfoundry/gosteno"
	"github.com/cloudfoundry/loggregatorlib/cfcomponent/instrumentation"
	"net"
	"sync/atomic"
)

const DefaultBufferSize = 4096

type LoggregatorClient interface {
	instrumentation.Instrumentable
	Send([]byte)
}

type udpLoggregatorClient struct {
	receivedMessageCount uint64
	sentMessageCount     uint64
	receivedByteCount    uint64
	sentByteCount        uint64
	sendChannel          chan []byte
	loggregatorAddress   string
}

func NewLoggregatorClient(loggregatorAddress string, logger *gosteno.Logger, bufferSize int) LoggregatorClient {
	loggregatorClient := &udpLoggregatorClient{}

	la, err := net.ResolveUDPAddr("udp", loggregatorAddress)
	if err != nil {
		logger.Fatalf("Error resolving loggregator address %s, %s", loggregatorAddress, err)
	}

	connection, err := net.ListenPacket("udp", "")
	if err != nil {
		logger.Fatalf("Error opening udp stuff")
	}

	loggregatorClient.loggregatorAddress = la.IP.String()
	loggregatorClient.sendChannel = make(chan []byte, bufferSize)

	go func() {
		for dataToSend := range loggregatorClient.sendChannel {
			if len(dataToSend) > 0 {
				writeCount, err := connection.WriteTo(dataToSend, la)
				if err != nil {
					logger.Errorf("Writing to loggregator %s failed %s", loggregatorAddress, err)
					continue
				}
				logger.Debugf("Wrote %d bytes to %s", writeCount, loggregatorAddress)
				atomic.AddUint64(&loggregatorClient.sentMessageCount, 1)
				atomic.AddUint64(&loggregatorClient.sentByteCount, uint64(writeCount))
			} else {
				logger.Debugf("Skipped writing of 0 byte message to %s", loggregatorAddress)
			}
		}
	}()

	return loggregatorClient
}

func (loggregatorClient *udpLoggregatorClient) Send(data []byte) {
	atomic.AddUint64(&loggregatorClient.receivedMessageCount, 1)
	atomic.AddUint64(&loggregatorClient.receivedByteCount, uint64(len(data)))
	loggregatorClient.sendChannel <- data
}

func (loggregatorClient *udpLoggregatorClient) metrics() []instrumentation.Metric {
	tags := map[string]interface{}{"loggregatorAddress": loggregatorClient.loggregatorAddress}
	return []instrumentation.Metric{
		instrumentation.Metric{Name: "currentBufferCount", Value: uint64(len(loggregatorClient.sendChannel)), Tags: tags},
		instrumentation.Metric{Name: "sentMessageCount", Value: atomic.LoadUint64(&loggregatorClient.sentMessageCount), Tags: tags},
		instrumentation.Metric{Name: "receivedMessageCount", Value: atomic.LoadUint64(&loggregatorClient.receivedMessageCount), Tags: tags},
		instrumentation.Metric{Name: "sentByteCount", Value: atomic.LoadUint64(&loggregatorClient.sentByteCount), Tags: tags},
		instrumentation.Metric{Name: "receivedByteCount", Value: atomic.LoadUint64(&loggregatorClient.receivedByteCount), Tags: tags},
	}
}

func (loggregatorClient *udpLoggregatorClient) Emit() instrumentation.Context {
	return instrumentation.Context{Name: "loggregatorClient",
		Metrics: loggregatorClient.metrics(),
	}
}
