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
	"github.com/petar/GoTeleport/tele/codec"
	"github.com/petar/GoTeleport/tele/trace"
	"net"
)

type Listener struct {
	frame trace.Frame
	sub   *codec.Listener
	ach   chan *Conn
}

func NewListener(frame trace.Frame, sub *codec.Listener) *Listener {
	l := &Listener{frame: frame, sub: sub, ach: make(chan *Conn)}
	frame.Bind(l)
	go func() {
		for {
			NewAcceptBridge(l.frame.Refine("accept"), l, l.sub.Accept())
		}
	}()
	return l
}

func (l *Listener) accept(conn *Conn) {
	l.ach <- conn
}

func (l *Listener) Accept() *Conn {
	return <-l.ach
}

func (l *Listener) Addr() net.Addr {
	return l.sub.Addr()
}
