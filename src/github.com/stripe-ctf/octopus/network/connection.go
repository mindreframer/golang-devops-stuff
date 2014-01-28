package network

import (
	"github.com/stripe-ctf/octopus/state"
	"log"
	"net"
	"time"
)

// There are two buffers, each of this size. I guess it's comparable to the
// kernel's send/recv queues.
const INFLIGHT = 15

type connection struct {
	link *Link
	kill chan bool
	// Two half-duplex streams, one per direction
	forward, backward *stream
	// The actual sockets at each end of the connection
	source, dest *fullDuplex
	// Ensure we clean up once reading/writing are complete
	closeRead, closeWrite chan bool
}

func (conn *connection) establish(l *label, dest string) {
	state.RecordConn()

	conn.forward = conn.stream(l)
	conn.backward = conn.stream(l.reverse())
	go conn.forward.start()
	go conn.backward.start()

	conn.forward.emptyPacket()
	go func() {
		select {
		case <-conn.kill:
			<-time.After(conn.link.Delay())
			conn.source.Close()
			return
		case <-conn.forward.out:
		}
		var err error
		d, err := net.Dial("unix", dest)
		conn.dest = NewSockWrap(d)
		if err != nil {
			log.Printf("establish %s: %s (hint: if this is a mysterious 'no such file or directory', that likely just means you need to wait for Octopus to finish setting up its sockets)", l, err)
			conn.source.Close()
			return
		}
		go conn.forward.writeTo(conn.dest)
		go conn.backward.readFrom(conn.dest)
	}()

	go conn.forward.readFrom(conn.source)
	go conn.backward.writeTo(conn.source)
}

func (conn *connection) stream(l *label) *stream {
	return &stream{
		l,
		conn,
		make(chan packet, INFLIGHT),
		make(chan []byte, INFLIGHT),
	}
}
