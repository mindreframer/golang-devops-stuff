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

package faithful

import (
	"net"

	"github.com/petar/GoTeleport/tele/chain"
	"github.com/petar/GoTeleport/tele/trace"
)

// Transport
type Transport struct {
	frame trace.Frame
	chain *chain.Transport
}

func NewTransport(frame trace.Frame, chain *chain.Transport) *Transport {
	t := &Transport{frame: frame, chain: chain}
	t.frame.Bind(t)
	return t
}

func (t *Transport) Listen(addr net.Addr) *Listener {
	return NewListener(t.frame.Refine("listener"), t.chain.Listen(addr))
}

// Dial returns instanteneously (it does not wait on I/O operations) and always succeeds,
// returning a non-nil connection object.
func (t *Transport) Dial(addr net.Addr) *Conn {
	conn := t.chain.Dial(addr)
	return NewConn(t.frame.Refine("dial"), conn)
}

// Listener
type Listener struct {
	frame trace.Frame
	sub   *chain.Listener
}

func NewListener(f trace.Frame, sub *chain.Listener) *Listener {
	l := &Listener{frame: f, sub: sub}
	l.frame.Bind(l)
	return l
}

func (l *Listener) Addr() net.Addr {
	return l.sub.Addr()
}

func (l *Listener) Accept() *Conn {
	return NewConn(l.frame.Refine("accept"), l.sub.Accept())
}
