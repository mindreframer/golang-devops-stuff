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
	"os"

	"github.com/petar/GoTeleport/tele"
	"github.com/petar/GoTeleport/tele/blend"
	"github.com/petar/GoTeleport/tele/tcp"
	"github.com/petar/GoTeleport/tele/trace"
)

// Client
type Client struct {
	frame   trace.Frame
	tele    *blend.Transport
	inAddr  string
	outAddr string
}

func NewClient(inAddr, outAddr string) {
	cli := &Client{frame: trace.NewFrame("tele", "client"), outAddr: outAddr}
	cli.frame.Bind(cli)

	// Make teleport transport
	t := tele.NewStructOverTCP()

	// Listen on input TCP address
	l, err := net.Listen("tcp", inAddr)
	if err != nil {
		cli.frame.Printf("listen on teleport address %s (%s)", inAddr, err)
		os.Exit(1)
	}
	if inAddr == "" {
		inAddr = l.Addr().String()
		fmt.Println(inAddr)
	}
	cli.tele, cli.inAddr = t, inAddr
	go cli.loop(l)
	return
}

func (cli *Client) loop(l net.Listener) {
	for {
		inConn, err := l.Accept()
		if err != nil {
			cli.frame.Printf("accept on tcp address %s (%s)", cli.inAddr, err)
			os.Exit(1)
		}
		// Contact teleport server
		tele := cli.tele.Dial(tcp.Addr(cli.outAddr))
		// Write an empty chunk to mark the beginning of connection
		if err = tele.Write(&cargo{}); err != nil {
			cli.frame.Printf("first write (%s)", err)
			tele.Close()
			inConn.Close()
			continue
		}
		// Begin proxying
		Proxy(inConn, tele)
	}
}
