// Copyright 2013 Petar Maymounkov
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package faithful provides a lossless chunked connection over a lossy one.
package faithful

import (
	"io"
	"net"
	"sync"
	"time"

	"github.com/petar/GoTeleport/tele/chain"
	"github.com/petar/GoTeleport/tele/trace"
)

type Conn struct {
	frame trace.Frame
	sub   *chain.Conn
	bfr   *Buffer          // Buffer is self-synchronized
	rch   chan interface{} // []byte or error
	sch   chan *control    // Sync write channel; readLoop sends and closes, syncWriteLoop receives
	ach   chan struct{}    // Abort channel
	// readLoop
	nread SeqNo // Number of received chunks
	nackd SeqNo // Number of sent and acknowledged chunks
	// Write
	wz__ sync.Mutex // Linearizes Write requests
}

type control struct {
	Writer *chain.ConnWriter
	Msg    encoder
}

func NewConn(frame trace.Frame, under *chain.Conn) *Conn {
	c := &Conn{
		frame: frame,
		// Only 1 needed in readLoop; get 3 just to be safe
		rch: make(chan interface{}, 3),
		// Capacity 1 (below) unblocks writeSync, invoked in readLoop right after a connection stitch
		// stitch is received, racing with a user write waiting on waitForLink.
		// In particular, if execUserWrite is waiting on waitForLink, it would prevent readLoop from
		// moving on to adopt the new connection.
		sch: make(chan *control, 1),
		ach: make(chan struct{}),
		sub: under,
		bfr: NewBuffer(frame.Refine("buffer"), MemoryCap),
	}
	c.frame.Bind(c)
	go c.readLoop()
	go c.writeLoop() // write loop for user and sync messages
	return c
}

func (c *Conn) Debug() interface{} {
	return c.sub.Debug()
}

const (
	MemoryCap      = 40
	AckFrequency   = 20
	LingerDuration = 60 * time.Second
)

// RemoteAddr returns the address of the remote endpoint on this faithful connection.
func (c *Conn) RemoteAddr() net.Addr {
	return c.sub.RemoteAddr()
}

// ===================================== READING ==========================================

// Read returns the next chunk received on the connection.
// chunk is non-nil if and only if err is nil.
func (c *Conn) Read() (chunk []byte, err error) {
	chunkOrErr, ok := <-c.rch
	if !ok {
		return nil, io.ErrUnexpectedEOF
	}
	chunk, ok = chunkOrErr.([]byte)
	if ok {
		return chunk, nil
	}
	return nil, chunkOrErr.(error)
}

func (c *Conn) syncWrite(writer *chain.ConnWriter, nackd SeqNo) {
	c.sch <- &control{Writer: writer, Msg: &Sync{NAckd: nackd}}
}

func (c *Conn) ackWrite(nackd SeqNo) {
	c.sch <- &control{Writer: nil, Msg: &Ack{NAckd: nackd}}
}

func (c *Conn) readLoop() {
	for {
		chunk, err := c.read()
		if err != nil {
			c.frame.Printf("read terminating: (%s)", err)
			c.rch <- err // rch must have buffer cap for this final error in case no one is Reading
			break
		}
		c.rch <- chunk
	}
	// Permanent connection end
	close(c.rch)
	close(c.sch)
	c.bfr.Abort()
}

// read blocks until the next CHUNK is received. Meanwhile it processes incoming control messages like ACK and SYNC.
// Non-nil errors returned by read indicate irrecoverable physical errors on the underlying connection.
func (c *Conn) read() ([]byte, error) {
	for {
		// Check for abort signal
		select {
		case <-c.ach:
			return nil, io.ErrUnexpectedEOF
		default:
		}
		// The read cannot block on anything else other than reading on the underlying connection.
		// Note that sub.Kill will unblock sub.Read. The former will be called by writeLoop when
		// it receives the abortion signal.
		chunk, err := c.sub.Read()

		// Stitching or permanent error
		if err != nil {
			if chunk != nil {
				panic("eh")
			}
			stitchConnWriter := chain.IsStitch(err)
			// Connection termination.
			// Any non-ErrStitch error implies connection termination.
			if stitchConnWriter == nil {
				c.frame.Println("read carrier error:", err.Error())
				return nil, err
			}
			// Stitching
			c.syncWrite(stitchConnWriter, c.nread)
			continue
		}

		// Payload received
		msg, err := decodeMsg(chunk)
		if err != nil {
			c.frame.Println("read/decode error:", err.Error())
			// Misbehaved opponent is a connection termination.
			return nil, err
		}
		switch t := msg.(type) {
		case *Sync:
			if err = c.readSync(t); err != nil {
				c.frame.Println("read SYNC error:", err.Error())
				// Permanent connection termination.
				return nil, err
			}
			// Retry read

		case *Ack:
			if err = c.readAck(t); err != nil {
				c.frame.Println("read ACK error:", err.Error())
				// Permanent connection termination.
				return nil, err
			}
			// Retry read

		case *Chunk:
			if chunk := c.readChunk(t); chunk != nil {
				return chunk, nil
			}
			// Redundant chunk was dropped. Retry read

		default:
			panic("eh")
		}
	}
}

// readChunk returns a non-nil chunk, if successful.
// Otherwise nil is returned to indicate that the packet was discarded.
func (c *Conn) readChunk(chunkMsg *Chunk) []byte {
	// Is this a chunk that was already received?
	// Drop already-received duplicates.
	if chunkMsg.seqno < c.nread {
		return nil
	}
	if chunkMsg.seqno == c.nread {
		c.nread++
		if c.nread%AckFrequency == 0 {
			c.ackWrite(c.nread)
		}
		if chunkMsg == nil {
			panic("eh")
		}
		return chunkMsg.chunk
	}
	// Otherwise, a future packet implies lost packets. Request a sync. Drop the future packet.
	c.syncWrite(nil, c.nread)
	return nil
}

func (c *Conn) readSync(syncMsg *Sync) error {
	//c.frame.Println("SYNC", syncMsg.NAckd)
	nackd := syncMsg.NAckd
	if nackd < c.nackd || nackd > c.bfr.NWritten() {
		return chain.ErrMisbehave
	}
	c.nackd = nackd
	// Seek before Remove, so that new chunks don't race into the network
	// redundantly as a result of Remove, before the old ones have been resent.
	c.bfr.Seek(nackd)
	c.bfr.Remove(nackd)
	return nil
}

func (c *Conn) readAck(ackMsg *Ack) error {
	//c.frame.Println("ACK", ackMsg.NAckd)
	nackd := ackMsg.NAckd
	if nackd < c.nackd || nackd > c.bfr.NWritten() {
		return chain.ErrMisbehave
	}
	c.nackd = nackd
	c.bfr.Remove(nackd)
	return nil
}

// ===================================== WRITING ==========================================

// Write blocks until the chunk is written to the connection.
// Write never returns an error, unless the connection is permanently broken.
func (c *Conn) Write(chunk []byte) (err error) {
	// Linearize user Writes & Closes
	c.wz__.Lock()
	defer c.wz__.Unlock()
	msg := &Chunk{chunk: chunk}
	return c.bfr.Write(msg)
}

// writeControl processes a single control message (SYNC or ACK).
// ok equals false if the connection is permanently broken and should be killed.
func (c *Conn) writeControl(writer *chain.ConnWriter, msg encoder) {
	chunk, err := msg.Encode()
	if err != nil {
		panic(err)
	}
	writer.Write(chunk)
}

// writeUser processes a single set of return values from Buffer.Read.
func (c *Conn) writeUser(writer *chain.ConnWriter, payload interface{}, seqno SeqNo, err error) (continueWriteLoop bool) {
	if err == io.EOF {
		// We've reached the EOF of the user write sequence.
		// Go back to listen for more reads from the buffer, in case a sync rewinds the buffer cursor.
		return true
	}
	if err == io.ErrUnexpectedEOF {
		// An unexpected termination has been reached, indicated by killing the buffer; nothing to send any longer.
		return false
	}
	if err != nil {
		panic("u")
	}
	// Encode chunk
	chunk := payload.(*Chunk)
	chunk.seqno = seqno // Sequence numbers are assigned 0-based integers
	raw, err := chunk.Encode()
	if err != nil {
		panic(err)
	}
	writer.Write(raw)
	// If connection is closed (no more writes) and the buffer is empty, it is time to kill the connection.
	if c.bfr.Drained() {
		return false
	}
	return true
}

func (c *Conn) writeLoop() {
	bfrch := NewBufferReadChan(c.bfr) // bfrchan returns a stream of chunks coming from buffer.Read
	defer func() {
		// Drain buffer Read until error
		for _ = range bfrch {
		}
		// Kill the underlying chain connection
		c.sub.Kill()
	}()
	var writer *chain.ConnWriter
	for {
		// If no writer available, wait for one from the readLoop
		if writer == nil {
			ctrl, ok := <-c.sch
			if !ok {
				return
			}
			if ctrl.Writer == nil {
				// Skip control messages that don't carry a new writer
				continue
			}
			writer = ctrl.Writer
			c.writeControl(writer, ctrl.Msg)
		}
		//
		select {
		case ctrl, ok := <-c.sch:
			if !ok {
				return
			}
			if ctrl.Writer != nil {
				writer = ctrl.Writer
			}
			c.writeControl(writer, ctrl.Msg)

		case user, ok := <-bfrch:
			if !ok {
				return
			}
			if !c.writeUser(writer, user.Payload, user.SeqNo, user.Err) {
				return
			}
		}
	}
}

// ===================================== CLOSING ==========================================

// Close closes the connection semantically. The connection object will linger for a short while
// to ensure that the closure event is delivered to the remote endpoint.
func (c *Conn) Close() (err error) {
	c.wz__.Lock()
	defer c.wz__.Unlock()
	c.bfr.Close()
	go func() {
		c.frame.Println("LINGER starting.")
		<-time.NewTimer(LingerDuration).C
		c.frame.Println("LINGER expired.")
		close(c.ach)
	}()
	return nil
}
