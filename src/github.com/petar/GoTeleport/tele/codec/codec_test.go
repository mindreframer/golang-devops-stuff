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

package codec

import (
	"os"
	"testing"
	"time"

	_ "circuit/kit/debug/ctrlc"
	"github.com/petar/GoTeleport/tele/chain"
	"github.com/petar/GoTeleport/tele/faithful"
	"github.com/petar/GoTeleport/tele/sandbox"
	"github.com/petar/GoTeleport/tele/trace"
)

type testMsg struct {
	Carry int
}

func failNow() {
	os.Exit(1)
}

const testN = 5

func TestCodec(t *testing.T) {

	// Transport
	f := trace.NewFrame()
	// Carrier
	sx := sandbox.NewRandomUnreliableTransport(f.Refine("sandbox"), 3, 3, time.Second/4, time.Second/4)
	// Chain
	hx := chain.NewTransport(f.Refine("chain"), sx)
	// Faithful
	fx := faithful.NewTransport(f.Refine("faithful"), hx)
	// Codec
	cx := NewTransport(fx, GobCodec{})

	// Sync
	y := make(chan int)

	// Accepter
	go func() {
		l := cx.Listen(sandbox.Addr("@"))
		for i := 0; i < testN; i++ {
			y <- 1
			conn := l.Accept()
			msg := &testMsg{}
			if err := conn.Read(msg); err != nil {
				t.Fatalf("read (%s)", err)
				failNow()
			}
			if msg.Carry != i {
				t.Fatalf("check")
				failNow()
			}
			f.Printf("READ %d/%d CLOSING", i+1, testN)
			conn.Close()
			f.Printf("READ %d/%d √", i+1, testN)
		}
		y <- 1
	}()

	// Dialer
	for i := 0; i < testN; i++ {
		<-y
		conn := cx.Dial(sandbox.Addr("@"))
		if err := conn.Write(&testMsg{i}); err != nil {
			t.Fatalf("write (%s)", err)
			failNow()
		}
		f.Printf("WRITE %d/%d CLOSING", i+1, testN)
		if err := conn.Close(); err != nil {
			t.Fatalf("close (%s)", err)
			failNow()
		}
		f.Printf("WRITE %d/%d √", i+1, testN)
	}
	<-y
}
