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
	"io"
	"net"
	"sync"

	"github.com/petar/GoTeleport/tele/trace"
)

/*
	Kill (turns Accept into a nop)
	  |
	  V
	Accept -- ch --> Link

*/

type acceptConn struct {
	Conn
	ach      chan *accept
	accept__ sync.Mutex
	closed   bool
	link__   sync.Mutex
	seqno    SeqNo // Sequence number of the current underlying connection
	carrier  net.Conn
}

type accept struct {
	Carrier net.Conn
	R       *bufio.Reader
	SeqNo   SeqNo
}

func newAcceptConn(frame trace.Frame, id chainID, addr net.Addr, carrier net.Conn, r *bufio.Reader, scrb func()) *acceptConn {
	ac := &acceptConn{
		ach: make(chan *accept, MaxHandshakes+3),
	}
	ac.Conn.Start(frame, id, addr, (*acceptLink)(ac), scrb)
	ac.Accept(carrier, r, 1)
	return ac
}

func (ac *acceptConn) Accept(carrier net.Conn, r *bufio.Reader, seqno SeqNo) {
	ac.accept__.Lock()
	defer ac.accept__.Unlock()
	if ac.closed {
		carrier.Close()
		return
	}
	ac.ach <- &accept{carrier, r, seqno}
}

type acceptLink acceptConn

// Kill shuts down the acceptLink, interrupting a pending wait for connection in Link.
func (al *acceptLink) Kill() {
	al.accept__.Lock()
	defer al.accept__.Unlock()
	if al.closed {
		return
	}
	al.closed = true
	close(al.ach)
}

// Link blocks until a new connection to the remote endpoint is passed through Accept or Kill is invoked.
func (al *acceptLink) Link(reason error) (net.Conn, *bufio.Reader, SeqNo, error) {
	al.link__.Lock()
	defer al.link__.Unlock()
	if al.carrier != nil {
		al.carrier.Close()
	}
	for {
		replaceWith, ok := <-al.ach
		if !ok {
			return nil, nil, 0, io.ErrUnexpectedEOF
		}
		if replaceWith.SeqNo > al.seqno {
			al.seqno = replaceWith.SeqNo
			al.carrier = replaceWith.Carrier
			al.Conn.frame.Printf("ACCEPTED #%d", al.seqno)
			return al.carrier, replaceWith.R, al.seqno, nil
		}
		al.Conn.frame.Printf("out-of-order redial #%d arrived while using #%d", replaceWith.SeqNo, al.seqno)
		replaceWith.Carrier.Close()
	}
	panic("u")
}
