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

	"github.com/petar/GoTeleport/tele/carrier"
	"github.com/petar/GoTeleport/tele/trace"
)

type Transport struct {
	frame   trace.Frame
	carrier carrier.Transport
	*Dialer
}

func NewTransport(frame trace.Frame, carrier carrier.Transport) *Transport {
	t := &Transport{
		frame:   frame,
		carrier: carrier,
		Dialer:  NewDialer(frame, carrier),
	}
	t.frame.Bind(t)
	return t
}

func (t *Transport) Listen(addr net.Addr) *Listener {
	return NewListener(t.frame.Refine("listener"), t.carrier, addr)
}
