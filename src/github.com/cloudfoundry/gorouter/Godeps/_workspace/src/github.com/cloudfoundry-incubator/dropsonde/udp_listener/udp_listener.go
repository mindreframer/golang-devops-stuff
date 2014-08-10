package udp_listener

import (
	"fmt"
	"net"
	"sync/atomic"
)

var UdpListeningPort = &port{42420}

func Run(dataChan chan<- []byte, stopChan <-chan struct{}) error {
	conn, err := net.ListenPacket("udp", fmt.Sprintf(":%d", UdpListeningPort.Get()))
	if err != nil {
		return err
	}
	UdpListeningPort.Set(conn.LocalAddr().(*net.UDPAddr).Port)

	go func() {
		<-stopChan
		conn.Close()
	}()

	for {
		buffer := make([]byte, 4096)
		var n int
		n, _, err = conn.ReadFrom(buffer)
		if err != nil {
			select {
			case <-stopChan:
				err = nil
			default:
			}

			return err
		}

		select {
		case dataChan <- buffer[0:n]:
		case <-stopChan:
			return nil
		}
	}
}

type port struct{ val int32 }

func (p *port) Set(val int) {
	atomic.StoreInt32(&p.val, int32(val))
}

func (p *port) Get() int {
	return int(atomic.LoadInt32(&p.val))
}
