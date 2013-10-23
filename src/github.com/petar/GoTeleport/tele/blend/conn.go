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
	"io"
	"sync"
)

/*

	 accept/dial            Conn
	+-----------+        +--------+
	|  pollread |------->| prompt |
	|    ···    |        |  ···   |
	|   write   |<-------| Write  |
	|   scrub   |<-------| Close  |
	+-----------+        +--------+

	Invariant "close-and-prompt": The connection object should not be
	registered with the AcceptConn as open, after the user has called Close,
	the AcceptConn has invoked Conn.prompt with an error.

	SOURCES OF CLOSURE:

	(nil,err) -----> prompt
	                  ···
	                 Close <--- USER
                      ···
	  write |<-----| Write
	        |-err->|

*/

type Conn struct {
	connID   ConnID
	bus      bus
	p__      sync.Mutex // send-side of prompt channel
	pch      chan *readReturn
	peof     bool // Prompt-side closure
	w__      sync.Mutex
	nwritten SeqNo // Number of writes
	weof     bool  // Write-side closure
}

type bus interface {
	write(*Msg) error
	scrub(ConnID)
}

type readReturn struct {
	Payload interface{}
	Err     error
}

func newConn(connID ConnID, bus bus) *Conn {
	return &Conn{connID: connID, bus: bus, pch: make(chan *readReturn, 3)}
}

// Read reads the next chunk of bytes.
func (c *Conn) Read() (interface{}, error) {
	rr, ok := <-c.pch
	if !ok {
		return nil, io.ErrUnexpectedEOF
	}
	return rr.Payload, rr.Err
}

func (c *Conn) prompt(payload interface{}, err error) {
	c.p__.Lock()
	defer c.p__.Unlock()
	if c.peof {
		return
	}
	c.pch <- &readReturn{Payload: payload, Err: err}
	if err != nil {
		close(c.pch)
		c.peof = true
	}
}

func (c *Conn) promptClose() {
	c.p__.Lock()
	defer c.p__.Unlock()
	if c.peof {
		return
	}
	close(c.pch)
	c.peof = true
}

// Write writes the chunk to the connection.
func (c *Conn) Write(v interface{}) error {
	c.w__.Lock()
	defer c.w__.Unlock()
	if c.weof {
		panic("writin after close")
	}
	c.nwritten++
	msg := &Msg{
		ConnID: c.connID,
		Demux: &PayloadMsg{
			SeqNo:   c.nwritten - 1,
			Payload: v,
		},
	}
	return c.bus.write(msg)
}

// Close closes the connection. It is synchronized with Write and will not interrupt a concurring write.
func (c *Conn) Close() error {
	c.w__.Lock()
	if c.weof {
		c.w__.Unlock()
		return io.ErrUnexpectedEOF
	}
	c.weof = true
	c.w__.Unlock()
	c.bus.scrub(c.connID) // Scrub outside of w__ lock
	c.promptClose()
	return nil
}
