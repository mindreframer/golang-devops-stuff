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
	"math/rand"
	"net"
	"time"

	"github.com/petar/GoTeleport/tele/trace"
)

func NewReliableTransport(f trace.Frame) *Transport {
	return NewTransport(f, NewPipe)
}

func NewUnreliableTransport(f trace.Frame, nok, ndrop int, expa, expb time.Duration) *Transport {
	return NewTransport(f, func(f0, f1 trace.Frame, a0, a1 net.Addr) (net.Conn, net.Conn) {
		f.Printf("TRANSPORT PROFILE NOK=%d, NDROP=%d", nok, ndrop)
		return NewSievePipe(f0, f1, a0, a1, nok, ndrop, expa, expb)
	})
}

func NewRandomUnreliableTransport(f trace.Frame, nok, ndrop int, expa, expb time.Duration) *Transport {
	return NewTransport(f, func(f0, f1 trace.Frame, a0, a1 net.Addr) (net.Conn, net.Conn) {
		nok, ndrop := rand.Intn(nok+1), rand.Intn(ndrop+1)
		nok = max(nok, 1)
		f.Printf("TRANSPORT PROFILE NOK=%d, NDROP=%d", nok, ndrop)
		return NewSievePipe(f0, f1, a0, a1, nok, ndrop, expa, expb)
	})
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}
