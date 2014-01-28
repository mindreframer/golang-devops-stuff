package network

import (
	"github.com/stripe-ctf/octopus/log"
	"github.com/stripe-ctf/octopus/state"
	"io"
	"time"
)

type packet struct {
	delay <-chan time.Time
	data  []byte
}

type stream struct {
	l    *label
	conn *connection
	in   chan packet
	out  chan []byte
}

// This goroutine handles copying packets between the in and out chans with
// added latency.
func (stream *stream) start() {
	for {
		var packet packet
		select {
		case <-stream.conn.kill:
			return
		case packet = <-stream.in:
		}
		<-packet.delay
		select {
		case <-stream.conn.kill:
			return
		case stream.out <- packet.data:
		}
	}
}

// We use empty packets for two things:
// 1. On the initial connection, we use it as a sentinel value to simulate the
//    latency of the initial connection. When the sentinel makes it all the way
//    through the stream, we actually attempt the connection.
// 2. When a connection is closed, an empty packet is placed into the queue.
//    When it arrives at the other end, we close the stream.
func (stream *stream) emptyPacket() {
	packet := packet{
		time.After(stream.conn.link.Delay()),
		nil,
	}
	// We need to make sure this doesn't block if the connection is killed
	// at an inopportune time
	select {
	case stream.in <- packet:
	case <-stream.conn.kill:
	}
}

func (stream *stream) readFrom(sock *fullDuplex) {
	for {
		// TODO: don't always chunk off exactly 1024 bytes
		buf := make([]byte, 1024)
		n, err := sock.Read(buf)
		state.RecordRead(n)
		if err != nil {
			if err != io.EOF {
				log.Debugf("readFrom %s: %s", stream.l, err)
			}
			sock.CloseRead()
			stream.emptyPacket()
			return
		}
		packet := packet{
			time.After(stream.conn.link.Delay()),
			buf[:n],
		}

		select {
		case stream.in <- packet:
		case <-stream.conn.kill:
			stream.closeLater(sock)
			return
		}
	}
}

func (stream *stream) writeTo(sock *fullDuplex) {
	for {
		var buf []byte
		select {
		case <-stream.conn.kill:
			stream.closeLater(sock)
			return
		case buf = <-stream.out:
		}
		// This is the special "close" notification
		if buf == nil {
			sock.CloseWrite()
			return
		}

		n, err := sock.Write(buf)
		state.RecordWrite(n)
		log.Debugf("Wrote %s %d bytes: %#v", stream.l, n, string(buf))
		if err != nil || n != len(buf) {
			log.Printf("writeTo %s: %s", stream.l, err)
			break
		}
	}

	// Close the other end of the connection
	time.Sleep(stream.conn.link.Delay())
	stream.other(sock).CloseRead()
}

func (stream *stream) other(sock *fullDuplex) *fullDuplex {
	if sock == stream.conn.source {
		return stream.conn.dest
	} else {
		return stream.conn.source
	}
}

func (stream *stream) closeLater(sock *fullDuplex) {
	time.Sleep(stream.conn.link.Delay())
	sock.Close()
}
