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
	"log"
	"reflect"
	"testing"

	_ "circuit/kit/debug/ctrlc"
	"github.com/petar/GoTeleport/tele/tcp"
	"github.com/petar/GoTeleport/tele/trace"
)

func TestConnOverTCP(t *testing.T) {
	frame := trace.NewFrame()
	x := tcp.Transport

	ready := make(chan int, 2)
	sent, recv := make(map[byte]struct{}), make(map[byte]struct{})

	// Accepter endpoint
	go func() {
		l := NewListener(frame.Refine("listener"), x, tcp.Addr(":17222"))
		ready <- 1 // SYNC: Notify that listener is accepting
		testGreedyRead(t, l.Accept(), recv)
		ready <- 1
	}()

	// Dialer endpoint
	<-ready // SYNC: Wait for listener to start accepting
	d := NewDialer(frame.Refine("dialer"), x)
	conn := d.Dial(tcp.Addr("localhost:17222"))
	testGreedyWrite(t, conn, sent)
	<-ready // SYNC: Wait for accepter goroutine to complete

	// Make sure all marked writes have been received
	if !reflect.DeepEqual(sent, recv) {
		t.Errorf("expected %#v, got %#v", sent, recv)
		failNow()
	}
}

func testGreedyRead(t *testing.T, conn *Conn, recv map[byte]struct{}) {
	var i int
	for i < testN {
		v, err := conn.Read()
		if err != nil {
			if v != nil {
				panic("eh")
			}
			// If not a stitching error, connection killed
			if IsStitch(err) == nil {
				return
			}
			log.Println("READ STITCH")
			continue
		}
		log.Printf("READ %d", v[0])
		recv[byte(v[0])] = struct{}{}
		i++
	}
	conn.Kill()
	log.Println("READ KILLED")
}

const testN = 5

func testGreedyWrite(t *testing.T, conn *Conn, sent map[byte]struct{}) {
	var i int
	for i < testN {
		_, err := conn.Read()
		w := IsStitch(err)
		if w == nil {
			log.Printf("non-stitch read on writer side")
			failNow()
		}
		for i < testN {
			log.Printf("WRITE %d", i)
			err := w.Write([]byte{byte(i)})
			if err != nil {
				t.Errorf("write (%s)", err)
				failNow()
			}
			sent[byte(i)] = struct{}{}
			i++
		}
	}
	conn.Kill()
	log.Println("WRITE KILLED")
}
