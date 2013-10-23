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

package main

import (
	"fmt"
	"net"
	"time"

	"github.com/petar/GoTeleport/tele"
	"github.com/petar/GoTeleport/tele/blend"
	"github.com/petar/GoTeleport/tele/tcp"
	"github.com/petar/GoTeleport/tele/trace"
)

// Server
type Server struct {
	frame   trace.Frame
	tele    *blend.Transport
	inAddr  string
	outAddr string
}

func NewServer(inAddr, outAddr string) (*Server, error) {
	t := tele.NewStructOverTCP()
	l := t.Listen(tcp.Addr(outAddr))
	if outAddr == "" {
		outAddr = l.Addr().String()
		fmt.Println(outAddr)
	}
	srv := &Server{frame: trace.NewFrame("tele", "server"), tele: t, inAddr: inAddr, outAddr: outAddr}
	srv.frame.Bind(srv)
	go srv.loop(l)
	return srv, nil
}

func (srv *Server) loop(l *blend.Listener) {
	for {
		outConn := l.Accept()
		// Read the first empty chunk from the connection
		if _, err := outConn.Read(); err != nil {
			srv.frame.Printf("first read (%s)", err)
			outConn.Close()
			continue
		}
		// Dial user server
		inConn, err := net.Dial("tcp", srv.inAddr)
		if err != nil {
			outConn.Close()
			srv.frame.Printf("server dial tcp address %s (%s)", srv.inAddr, err)
			time.Sleep(time.Second) // Prevents DoS when local TCP server is down temporarily
			continue
		}
		Proxy(inConn, outConn)
	}
}
