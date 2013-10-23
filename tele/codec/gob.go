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
	"bytes"
	"encoding/gob"
)

// GobCodec
type GobCodec struct{}

func (GobCodec) NewEncoder() Encoder {
	return NewGobEncoder()
}

func (GobCodec) NewDecoder() Decoder {
	return NewGobDecoder()
}

// GobEncoder
type GobEncoder struct {
	w   writer
	enc *gob.Encoder
}

func NewGobEncoder() *GobEncoder {
	g := &GobEncoder{}
	g.w.Clear()
	g.enc = gob.NewEncoder(&g.w)
	return g
}

func (g *GobEncoder) Encode(v interface{}) ([]byte, error) {
	if err := g.enc.Encode(v); err != nil {
		return nil, err
	}
	return g.w.Flush(), nil
}

// GobDecoder
type GobDecoder struct {
	r   reader
	dec *gob.Decoder
}

func NewGobDecoder() *GobDecoder {
	g := &GobDecoder{}
	g.dec = gob.NewDecoder(&g.r)
	return g
}

func (g *GobDecoder) Decode(p []byte, v interface{}) error {
	g.r.Load(p)
	return g.dec.Decode(v)
}

//
type writer struct {
	buf *bytes.Buffer
}

func (w *writer) Clear() {
	w.buf = new(bytes.Buffer)
}

func (w *writer) Write(p []byte) (int, error) {
	return w.buf.Write(p)
}

func (w *writer) Flush() []byte {
	defer func() {
		w.buf = new(bytes.Buffer)
	}()
	return w.buf.Bytes()
}

//
type reader struct {
	buf *bytes.Reader
}

func (r *reader) Load(p []byte) {
	r.buf = bytes.NewReader(p)
}

func (r *reader) Read(p []byte) (int, error) {
	return r.buf.Read(p)
}
