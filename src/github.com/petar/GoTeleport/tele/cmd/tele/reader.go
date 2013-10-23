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
	"encoding/gob"
)

func init() {
	gob.Register(&cargo{})
}

type cargo struct {
	Cargo []byte
}

/*
type faithfulReader struct {
	sync.Mutex
	conn *faithful.Conn
	buf  bytes.Buffer
}

func NewFaithfulReader(conn *faithful.Conn) *faithfulReader {
	return &faithfulReader{}
}

func (x *faithfulReader) Read(p []byte) (int, error) {
	x.Lock()
	defer x.Unlock()
	for x.buf.Len() == 0 {
		blob, err := x.conn.Read()
	}
	return x.buf.Read(p)
}
*/
