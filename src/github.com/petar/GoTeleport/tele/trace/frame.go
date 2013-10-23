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

// Package trace implements an ad-hoc tracing system
package trace

import (
	"bytes"
	"fmt"
	"log"
	"reflect"
	"unicode/utf8"
)

// Framed is an object that has a Frame
type Framed interface {
	Frame() Frame
}

// Frame …
type Frame interface {
	Refine(sub ...string) Frame
	Bind(interface{})

	Println(v ...interface{})
	Print(v ...interface{})
	Printf(format string, v ...interface{})
	String() string
	Chain() []string
}

// frame implements Frame
type frame struct {
	ptr uintptr
	chain
}

func NewFrame(s ...string) Frame {
	return &frame{chain: chain(s)}
}

func (f *frame) Refine(sub ...string) Frame {
	c := make(chain, len(f.chain), len(f.chain)+len(sub))
	copy(c, f.chain)
	c = append(c, sub...)
	return &frame{chain: c}
}

func (f *frame) Bind(v interface{}) {
	if f.ptr != 0 {
		panic("duplicate binding")
	}
	f.ptr = reflect.ValueOf(v).Pointer()
}

func justify(s string, l int) string {
	var w bytes.Buffer
	n := utf8.RuneCountInString(s)
	for i := 0; i < max(0, l-n); i++ {
		w.WriteRune('·')
	}
	w.WriteString(s)
	return string(w.Bytes())
}

func max(x, y int) int {
	if x > y {
		return x
	}
	return y
}

func min(x, y int) int {
	if x < y {
		return x
	}
	return y
}

func (f *frame) String() string {
	return fmt.Sprintf("(0x%010x) |%s|", f.ptr, justify(f.chain.String(), 60))
}

func (f *frame) Println(v ...interface{}) {
	log.Println(append([]interface{}{f.String()}, v...)...)
}

func (f *frame) Print(v ...interface{}) {
	log.Print(append([]interface{}{f.String()}, v...)...)
}

func (f *frame) Printf(format string, v ...interface{}) {
	log.Printf("%s "+format, append([]interface{}{f.String()}, v...)...)
}

func (f *frame) Chain() []string {
	return []string(f.chain)
}

// chain is an ordered sequence of strings with a String method
type chain []string

func (c chain) String() string {
	var w bytes.Buffer
	for i, s := range c {
		if i > 0 {
			w.WriteString("—>")
		}
		w.WriteString(s)
	}
	return string(w.Bytes())
}
