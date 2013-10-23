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

// Package sandbox provides a simulated carrier Transport for testing purposes.
package sandbox

import (
	"net"
	"sync"

	"github.com/petar/GoTeleport/tele/carrier"
	"github.com/petar/GoTeleport/tele/trace"
)

// Addr is a sandbox address, implementing net.Addr
type Addr string

func (a Addr) Network() string {
	return "sandbox"
}

func (a Addr) String() string {
	return string(a)
}

// Transport implements a sandbox-ed internetworking infrastructure with a customizable link behavior.
type Transport struct {
	frame trace.Frame
	connMaker
	sync.Mutex
	withAddr map[string]*listener
}

// PipeMaker is a function that creates a pipe between two addresses.
type connMaker func(af, bf trace.Frame, a, b net.Addr) (net.Conn, net.Conn)

// New creates a new sandboxed network with connections supplied by piper.
func NewTransport(f trace.Frame, connMaker connMaker) *Transport {
	return &Transport{
		frame:     f,
		connMaker: connMaker,
		withAddr:  make(map[string]*listener),
	}
}

func (s *Transport) Frame() trace.Frame {
	return s.frame
}

// Listen starts a new listener object at the given opaque address.
func (s *Transport) Listen(addr net.Addr) (net.Listener, error) {
	s.Lock()
	defer s.Unlock()
	l, ok := s.withAddr[addr.String()]
	if !ok {
		l = newListener(s.frame.Refine("listener"), addr)
		s.withAddr[addr.String()] = l
	}
	return l, nil
}

// Dial dials the opaque address.
func (s *Transport) Dial(addr net.Addr) (net.Conn, error) {
	s.Lock()
	defer s.Unlock()
	l, ok := s.withAddr[addr.String()]
	if !ok {
		return nil, carrier.ErrPerm
	}
	p0, p1 := s.connMaker(
		s.frame.Refine("dial", addr.String()), s.frame.Refine("accept"),
		Addr(addr.String()), Addr(addr.String()),
	)
	l.connect(p1)
	return p0, nil
}
