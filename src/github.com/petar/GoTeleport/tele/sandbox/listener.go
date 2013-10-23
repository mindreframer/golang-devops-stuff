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

package sandbox

import (
	"io"
	"net"
	"sync"

	"github.com/petar/GoTeleport/tele/trace"
)

// listener implements a net.Listener for the sandbox Transport.
type listener struct {
	trace.Frame
	addr net.Addr
	ch__ sync.Mutex
	ch   chan net.Conn
}

func newListener(f trace.Frame, addr net.Addr) *listener {
	l := &listener{Frame: f, addr: addr, ch: make(chan net.Conn)}
	l.Frame.Bind(l)
	return l
}

func (sl *listener) connect(p net.Conn) {
	sl.ch__.Lock()
	defer sl.ch__.Unlock()
	sl.ch <- p
}

func (sl *listener) Accept() (net.Conn, error) {
	p, ok := <-sl.ch
	if !ok {
		return nil, io.ErrUnexpectedEOF
	}
	return p, nil
}

func (sl *listener) Close() error {
	sl.ch__.Lock()
	defer sl.ch__.Unlock()
	close(sl.ch)
	sl.ch = nil
	return nil
}

func (sl *listener) Addr() net.Addr {
	return sl.addr
}
