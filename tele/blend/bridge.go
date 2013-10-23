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

package blend

import (
	"io"
	"sync"

	"github.com/petar/GoTeleport/tele/codec"
	"github.com/petar/GoTeleport/tele/trace"
)

// AcceptBridge
type AcceptBridge struct {
	Bridge
}

func NewAcceptBridge(frame trace.Frame, l *Listener, sub *codec.Conn) *AcceptBridge {
	ab := &AcceptBridge{}
	ab.Init(frame, l, false, sub, nil)
	return ab
}

// DialBridge
type DialBridge struct {
	Bridge
}

func NewDialBridge(frame trace.Frame, sub *codec.Conn, scrb func()) *DialBridge {
	db := &DialBridge{}
	db.Init(frame, nil, true, sub, scrb)
	return db
}

// Bridge
type Bridge struct {
	frame    trace.Frame
	scrb     func()
	listener *Listener
	dialing  bool
	sub      *codec.Conn
	open__   sync.Mutex
	ndial    ConnID
	open     map[ConnID]*Conn
	wz__     sync.Mutex
}

func (b *Bridge) Init(frame trace.Frame, l *Listener, dialing bool, sub *codec.Conn, scrb func()) {
	b.frame = frame
	b.scrb = scrb
	b.frame.Bind(b)
	b.listener, b.dialing = l, dialing
	b.sub = sub
	b.open = make(map[ConnID]*Conn)
	go func() {
		defer b.teardown()
		for {
			if err := b.read(); err != nil {
				b.frame.Printf("blender read loop (%s)", err)
				return
			}
		}
	}()
}

func (b *Bridge) teardown() {
	b.open__.Lock()
	b.sub.Close()
	b.wz__.Lock()
	b.sub = nil
	b.wz__.Unlock()
	for connID, conn := range b.open {
		conn.promptClose()
		delete(b.open, connID)
	}
	b.open__.Unlock()
	if b.scrb != nil {
		b.scrb()
	}
}

func (b *Bridge) read() error {
	msg := &Msg{}
	if err := b.sub.Read(msg); err != nil {
		// Connection broken
		return err
	}

	switch t := msg.Demux.(type) {
	case *PayloadMsg:
		conn := b.get(msg.ConnID)
		if conn != nil {
			// Existing connection
			conn.prompt(t.Payload, nil)
			return nil
		}
		// Dead connection
		if t.SeqNo > 0 {
			b.writeClose(msg.ConnID)
			return nil
		}
		// New connection
		if b.listener != nil {
			conn = newConn(msg.ConnID, b)
			b.set(msg.ConnID, conn)
			conn.prompt(t.Payload, nil)
			b.listener.accept(conn) // Send new connection to user
			return nil
		} else {
			b.writeClose(msg.ConnID)
			return nil
		}

	case *CloseMsg:
		conn := b.get(msg.ConnID)
		if conn == nil {
			// Discard CLOSE for non-existent connections
			// Do not respond with a CLOSE packet. It would cause an avalanche of CLOSEs.
			return nil
		}
		b.scrub(msg.ConnID)
		conn.prompt(nil, io.EOF)
		return nil
	}

	// Unexpected remote behavior
	return ErrProto
}

func (b *Bridge) Dial() *Conn {
	b.open__.Lock()
	defer b.open__.Unlock()
	if b.open[b.ndial] != nil {
		panic("u")
	}
	conn := newConn(b.ndial, b)
	b.open[b.ndial] = conn
	b.ndial++
	return conn
}

func (b *Bridge) get(connID ConnID) *Conn {
	b.open__.Lock()
	defer b.open__.Unlock()
	return b.open[connID]
}

func (b *Bridge) set(connID ConnID, conn *Conn) {
	b.open__.Lock()
	defer b.open__.Unlock()
	b.open[connID] = conn
}

func (b *Bridge) scrub(connID ConnID) {
	b.open__.Lock()
	defer b.open__.Unlock()
	delete(b.open, connID)
}

func (b *Bridge) write(msg *Msg) error {
	b.wz__.Lock()
	defer b.wz__.Unlock()
	//
	if b.sub == nil {
		return io.ErrUnexpectedEOF
	}
	if err := b.sub.Write(msg); err != nil {
		return err
	}
	return nil
}

func (b *Bridge) writeClose(connID ConnID) error {
	msg := &Msg{
		ConnID: connID,
		Demux:  &CloseMsg{},
	}
	return b.write(msg)
}
