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
	"net"

	"github.com/petar/GoTeleport/tele/blend"
	"github.com/petar/GoTeleport/tele/trace"
)

type proxy struct {
	frame  trace.Frame
	legacy net.Conn
	tele   *blend.Conn
}

func Proxy(legacy net.Conn, tele *blend.Conn) {
	p := &proxy{frame: trace.NewFrame("proxy"), legacy: legacy, tele: tele}
	p.frame.Bind(p)
	go p.legacy2tele()
	go p.tele2legacy()
}

const ReadBlockLen = 1e4

func (p *proxy) legacy2tele() {
	var (
		n   int
		err error
	)
	// We avoid buffer creation on each iteration since blend.Write copies the data before it returns.
	buf := make([]byte, ReadBlockLen)
	for {
		n, err = p.legacy.Read(buf)
		if n > 0 {
			if err := p.tele.Write(&cargo{Cargo: buf[:n]}); err != nil {
				p.frame.Printf("write (%s)", err)
				p.tele.Close()
				return
			}
			continue
		}
		if err == nil {
			panic("e")
		}
		p.frame.Printf("read (%s)", err)
		p.tele.Close()
		return
	}
}

func (p *proxy) tele2legacy() {
	var (
		err   error
		chunk interface{}
	)
	for {
		chunk, err = p.tele.Read()
		if err != nil {
			p.frame.Printf("read (%s)", err)
			p.legacy.Close()
			return
		}
		if _, err = p.legacy.Write(chunk.(*cargo).Cargo); err != nil {
			p.frame.Printf("write (%s)", err)
			p.legacy.Close()
			return
		}
	}
}
