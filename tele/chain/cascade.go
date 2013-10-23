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
	"sync"

	"github.com/petar/GoTeleport/tele/trace"
)

type Cascade struct {
	frame trace.Frame
	rw    sync.Mutex  // Read/write lock
	v     interface{} // Current value
	i     int
	exp   chan struct{} // Expire broadcast for current value
}

// End-of-interval
type eoi struct{}

func MakeCascade(frame trace.Frame) *Cascade {
	return &Cascade{frame: frame, exp: make(chan struct{})}
}

// Close closes the cascade. All waiting receivers unblock.
func (x *Cascade) Close() {
	x.Transition(eoi{})
}

// Transition …
func (x *Cascade) Transition(v interface{}) {
	if v == nil {
		panic("cannot yield nil interfaces")
	}
	x.rw.Lock()
	defer x.rw.Unlock()
	if _, ok := x.v.(eoi); ok {
		panic("replace after close")
	}
	if x.exp != nil {
		close(x.exp)
	}
	x.v, x.i, x.exp = v, x.i+1, make(chan struct{})
}

func (x *Cascade) recv() (v interface{}, i int, expire <-chan struct{}, ok bool) {
	v, i, expire, ok = x.recvspin()
	if !ok {
		// Closed
		return
	}
	if v != nil {
		// Value available
		return
	}
	// If the first transition hasn't happened yet, wait for it.
	<-expire
	return x.recvspin()
}

func (x *Cascade) recvspin() (v interface{}, i int, expire <-chan struct{}, ok bool) {
	x.rw.Lock()
	defer x.rw.Unlock()
	// Check whether cascade is closed
	if _, ok := x.v.(eoi); ok {
		return nil, 0, nil, false
	}
	return x.v, x.i, x.exp, true
}

func (x *Cascade) Current() *CascadeInterval {
	v, i, exp, ok := x.recv()
	if !ok {
		return nil
	}
	return &CascadeInterval{x: x, i: i, v: v, exp: exp}
}

// CascadeInterval …
type CascadeInterval struct {
	x   *Cascade
	i   int
	v   interface{}
	exp <-chan struct{}
}

func (y *CascadeInterval) Value() interface{} {
	return y.v
}

func (y *CascadeInterval) Next() *CascadeInterval {
	<-y.exp
	return y.x.Current()
}

func (y *CascadeInterval) Expire() {
	<-y.exp
}
