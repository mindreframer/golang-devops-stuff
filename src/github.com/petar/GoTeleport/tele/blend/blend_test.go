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
	"encoding/gob"
	"os"
	"testing"
	"time"

	_ "circuit/kit/debug/ctrlc"
	"github.com/petar/GoTeleport/tele/chain"
	"github.com/petar/GoTeleport/tele/codec"
	"github.com/petar/GoTeleport/tele/faithful"
	"github.com/petar/GoTeleport/tele/sandbox"
	"github.com/petar/GoTeleport/tele/trace"
)

type testMsg struct {
	Carry int
}

func init() {
	gob.Register(&testMsg{})
}

func failNow() {
	os.Exit(1)
}

const testN = 100

func TestCodec(t *testing.T) {

	// Transport
	f := trace.NewFrame()
	// Carrier
	sx := sandbox.NewUnreliableTransport(f.Refine("sandbox"), 5, 0, time.Second/3, time.Second/3)
	// Chain
	hx := chain.NewTransport(f.Refine("chain"), sx)
	// Faithful
	fx := faithful.NewTransport(f.Refine("faithful"), hx)
	// Codec
	cx := codec.NewTransport(fx, codec.GobCodec{})
	// Blend
	bx := NewTransport(f.Refine("blend"), cx)

	// Sync
	ya, yb := make(chan int), make(chan int)

	// Accepter
	l := bx.Listen(sandbox.Addr("@"))
	for i := 0; i < testN; i++ {
		go testAcceptConn(t, l, ya, yb)
	}

	// Dialer
	for i := 0; i < testN; i++ {
		go testDialConn(t, bx, ya, yb)
	}
	for i := 0; i < testN; i++ {
		<-yb
	}
}

func testDialConn(t *testing.T, bx *Transport, ya, yb chan int) {
	<-ya
	conn := bx.Dial(sandbox.Addr("@"))
	if err := conn.Write(&testMsg{77}); err != nil {
		t.Errorf("write (%s)", err)
		failNow()
	}
	if err := conn.Close(); err != nil {
		t.Errorf("close (%s)", err)
		failNow()
	}
}

func testAcceptConn(t *testing.T, l *Listener, ya, yb chan int) {
	ya <- 1
	conn := l.Accept()
	msg, err := conn.Read()
	if err != nil {
		t.Errorf("read (%s)", err)
		failNow()
	}
	if msg.(*testMsg).Carry != 77 {
		t.Errorf("check")
		failNow()
	}
	conn.Close()
	yb <- 1
}
