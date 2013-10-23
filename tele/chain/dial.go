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
	"log"
	"math/rand"
	"net"
	"sync"
	"time"

	carrierPkg "github.com/petar/GoTeleport/tele/carrier"
	"github.com/petar/GoTeleport/tele/trace"
)

type dialConn struct {
	dial dialFunc
	Conn
	x__     sync.Mutex
	ndial   SeqNo // Number of redials so far
	carrier net.Conn
}

type dialFunc func() (net.Conn, error)

func newDialConn(frame trace.Frame, id chainID, addr net.Addr, dial dialFunc) *dialConn {
	dc := &dialConn{dial: dial}
	dc.Conn.Start(frame, id, addr, (*dialLink)(dc), nil)
	return dc
}

// dialLink is an alias for dialConn which implements the linker interface
type dialLink dialConn

// Link attempts to dial a new connection to the remote endpoint.
func (dl *dialLink) Link(reason error) (net.Conn, *bufio.Reader, SeqNo, error) {
	dl.x__.Lock()
	defer dl.x__.Unlock()

	if dl.carrier != nil {
		dl.carrier.Close()
	}
	if dl.carrier == nil && dl.ndial > 0 {
		return nil, nil, 0, io.ErrUnexpectedEOF
	}
	if dl.ndial > 0 {
		time.Sleep(CarrierRedialTimeout)
	}
	for {
		carrier, err := dl.dial()
		if err != nil {
			if err == carrierPkg.ErrPerm {
				// Permanent error on the carrier connections dial attempts means
				// do not retry.
				dl.frame.Printf("permanently unreachable")
				return nil, nil, 0, err
			}
			// Non-permanent errors result in redial.
			dl.frame.Printf("dial attempt (%s)", err)
			time.Sleep(CarrierRedialTimeout + time.Duration((int64(CarrierRedialTimeout)*rand.Int63n(1e6))/1e6))
			continue
		}
		if err = dl.handshake(carrier); err != nil {
			// All errors returned fromdl.handshake imply a broken carrier
			// connection. Here we defer reporting the broken connection error
			// to the Conn object, which will spot when it tries to use it.
			// This way, connection errors are treated more uniformly at the
			// same spot in the Conn logic. Furthermore, retrying from this
			// error condition here would break some test-only logic when the
			// underlying sandbox connection works in regime NOK=1 NDROP=0
			log.Printf("chain dial handshake (%s)", err)
		}
		dl.carrier = carrier
		return dl.carrier, bufio.NewReader(carrier), dl.ndial, nil
	}
	panic("u")
}

func (dl *dialLink) handshake(carrier net.Conn) (err error) {
	var seqno SeqNo
	dl.ndial, seqno = dl.ndial+1, dl.ndial+1 // Connection count starts from 1
	msg := &msgDial{ID: dl.Conn.id, SeqNo: seqno}
	defer dl.Conn.frame.Printf("DIALED #%d", seqno)
	return msg.Write(carrier)
}

// Kill shuts down the dialLink after a possibly concurring Link completes.
func (dl *dialLink) Kill() {
	dl.x__.Lock()
	defer dl.x__.Unlock()
	if dl.carrier == nil {
		return
	}
	dl.carrier.Close()
	dl.carrier = nil
}
