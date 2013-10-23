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
	"net"
	"time"

	"github.com/petar/GoTeleport/tele/carrier"
	"github.com/petar/GoTeleport/tele/trace"
)

// Dialer is a chunk.Dialer that establishes stitched chunk.Conns.
type Dialer struct {
	frame trace.Frame
	carrier.Transport
}

const CarrierRedialTimeout = time.Second

// NewDialer allocates a new chunk.Dialer for stitched chunk.Conns.
func NewDialer(frame trace.Frame, carrier carrier.Transport) *Dialer {
	return &Dialer{
		frame:     frame,
		Transport: carrier,
	}
}

// Dial creates a new chain connection to addr.
// Dial returns instanteneously (it does not wait on I/O operations) and always succeeds,
// returning a non-nil connection object.
func (d *Dialer) Dial(addr net.Addr) *Conn {
	dc := newDialConn(d.frame.Refine(addr.String()), chooseChainID(), addr,
		func() (net.Conn, error) {
			return d.Transport.Dial(addr)
		},
	)
	return &dc.Conn
}
