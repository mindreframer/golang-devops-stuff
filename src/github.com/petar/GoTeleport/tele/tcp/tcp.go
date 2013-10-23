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

// Package tcp implements a carrier transport over TCP.
package tcp

import (
	"net"
	"strings"

	"github.com/petar/GoTeleport/tele/carrier"
	"github.com/petar/GoTeleport/tele/trace"
)

// Transport is a carrier.Transport over TCP.
var Transport = transport{trace.NewFrame("tcp")}

type transport struct {
	trace.Frame
}

func (transport) Listen(addr net.Addr) (net.Listener, error) {
	t := addr.String()
	if strings.Index(t, ":") < 0 {
		t = t + ":0"
	}
	l, err := net.Listen("tcp", t)
	if err != nil {
		return nil, err
	}
	return listener{l}, nil
}

func (transport) Dial(addr net.Addr) (net.Conn, error) {
	c, err := net.Dial("tcp", addr.String())
	if err != nil {
		operr, ok := err.(*net.OpError)
		if !ok {
			return nil, err
		}
		if operr.Temporary() {
			return nil, err
		}
		return nil, carrier.ErrPerm
	}
	return &conn{trace.NewFrame("tcp", "dial"), c}, nil
}

type listener struct {
	net.Listener
}

func (l listener) Accept() (net.Conn, error) {
	c, err := l.Listener.Accept()
	if err != nil {
		return nil, err
	}
	return &conn{trace.NewFrame("tcp", "acpt"), c}, nil
}

type conn struct {
	trace.Frame
	net.Conn
}
