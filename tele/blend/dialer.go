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
	"net"
	"sync"

	"github.com/petar/GoTeleport/tele/codec"
	"github.com/petar/GoTeleport/tele/trace"
)

type Dialer struct {
	frame    trace.Frame
	sub      *codec.Transport
	bridge__ sync.Mutex
	bridge   map[string]*DialBridge
}

func NewDialer(frame trace.Frame, sub *codec.Transport) *Dialer {
	d := &Dialer{frame: frame, sub: sub, bridge: make(map[string]*DialBridge)}
	d.frame.Bind(d)
	return d
}

func (d *Dialer) Dial(addr net.Addr) *Conn {
	d.bridge__.Lock()
	defer d.bridge__.Unlock()
	db := d.bridge[addr.String()]
	if db == nil {
		conn := d.sub.Dial(addr) // codec.Dial always returns instantaneously.
		db = NewDialBridge(d.frame.Refine("dial"), conn, func() {
			d.scrub(addr.String())
		})
		d.bridge[addr.String()] = db
	}
	return db.Dial()
}

func (d *Dialer) scrub(addr string) {
	d.bridge__.Lock()
	defer d.bridge__.Unlock()
	delete(d.bridge, addr)
}
