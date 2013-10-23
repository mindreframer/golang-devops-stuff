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

package codec

import (
	"net"

	"github.com/petar/GoTeleport/tele/faithful"
)

type Transport struct {
	sub   *faithful.Transport
	codec Codec
}

func NewTransport(sub *faithful.Transport, codec Codec) *Transport {
	return &Transport{sub: sub, codec: codec}
}

// Dial returns instanteneously (it does not wait on I/O operations) and always succeeds,
// returning a non-nil connection object.
func (t *Transport) Dial(addr net.Addr) *Conn {
	conn := t.sub.Dial(addr)
	return NewConn(conn, t.codec)
}

func (t *Transport) Listen(addr net.Addr) *Listener {
	return &Listener{
		codec:    t.codec,
		Listener: t.sub.Listen(addr),
	}
}

type Listener struct {
	codec Codec
	*faithful.Listener
}

func (l *Listener) Accept() *Conn {
	return NewConn(l.Listener.Accept(), l.codec)
}

func (l *Listener) Addr() net.Addr {
	return l.Listener.Addr()
}
