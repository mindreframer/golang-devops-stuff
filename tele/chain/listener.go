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

package chain

import (
	"bufio"
	"net"
	"sync"

	"github.com/petar/GoTeleport/tele/limiter"
	"github.com/petar/GoTeleport/tele/carrier"
	"github.com/petar/GoTeleport/tele/trace"
)

// Listener
type Listener struct {
	frame    trace.Frame
	listener net.Listener
	withID__ sync.Mutex
	withID   map[chainID]*acceptConn
	accpt__  sync.Mutex
	accpt    chan *Conn
}

const (
	MaxHandshakes = 5 // Maximum number of concurrent handshake interactions
)

func NewListener(frame trace.Frame, carrier carrier.Transport, addr net.Addr) *Listener {
	cl, err := carrier.Listen(addr)
	if err != nil {
		panic(err)
	}
	l := &Listener{
		frame:    frame,
		listener: cl,
		withID:   make(map[chainID]*acceptConn),
		accpt:    make(chan *Conn),
	}
	l.frame.Bind(l)
	go l.loop()
	return l
}

func (l *Listener) Addr() net.Addr {
	return l.listener.Addr()
}

func (l *Listener) loop() {
	lmtr := limiter.New(MaxHandshakes)
	for {
		lmtr.Open()
		c, err := l.listener.Accept()
		if err != nil {
			lmtr.Close()
			panic(err) // Best not to be quiet about it
		}
		go func() {
			l.handshake(c)
			lmtr.Close()
		}()
	}
}

func (l *Listener) handshake(carrier net.Conn) {
	r := bufio.NewReader(carrier)
	dialMsg, err := readMsgDial(r)
	if err != nil && dialMsg != nil {
		panic("eh")
	}
	if err != nil {
		l.frame.Printf("Unrecognized dial message (%s) during chain connection accept handshake", err)
		carrier.Close()
		return
	}

	switch dialMsg.SeqNo {
	case 0:
		l.frame.Printf("Protocol error (connection seqeunce numbers start from 1)")
		carrier.Close()
		return

	case 1:
		var ac *acceptConn
		if ac, err = l.make(dialMsg.ID, carrier, r); err != nil {
			l.frame.Printf("Incoming duplicate chain connection (%s) %x", err, dialMsg.ID)
			carrier.Close()
			return
		}
		// Send connection for Accept, if not a re-dial
		l.accpt__.Lock()
		l.accpt <- &ac.Conn
		l.accpt__.Unlock()

	default:
		var ac *acceptConn
		if ac = l.get(dialMsg.ID); ac == nil {
			l.frame.Printf("Redial after a chain of connections %x has closed permanently", dialMsg.ID)
			carrier.Close()
			return
		}
		ac.Accept(carrier, r, dialMsg.SeqNo)
	}
}

func (l *Listener) Accept() *Conn {
	return <-l.accpt
}

func (l *Listener) make(id chainID, carrier net.Conn, r *bufio.Reader) (*acceptConn, error) {
	l.withID__.Lock()
	defer l.withID__.Unlock()
	ac, ok := l.withID[id]
	if ok {
		return ac, errDup
	}
	addr := carrier.RemoteAddr()
	ac = newAcceptConn(l.frame.Refine(l.Addr().String()), id, addr, carrier, r, func() {
		l.scrub(id)
	})
	l.withID[id] = ac
	return ac, nil
}

func (l *Listener) get(id chainID) *acceptConn {
	l.withID__.Lock()
	defer l.withID__.Unlock()
	return l.withID[id]
}

func (l *Listener) scrub(id chainID) *acceptConn {
	l.withID__.Lock()
	defer l.withID__.Unlock()
	a, ok := l.withID[id]
	if !ok {
		return nil
	}
	delete(l.withID, id)
	return a
}
