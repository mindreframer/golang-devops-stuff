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
	"github.com/petar/GoTeleport/tele/faithful"
)

type Conn struct {
	enc   Encoder
	dec   Decoder
	faith *faithful.Conn
}

func NewConn(faith *faithful.Conn, codec Codec) *Conn {
	return &Conn{
		enc:   codec.NewEncoder(),
		dec:   codec.NewDecoder(),
		faith: faith,
	}
}

func (c *Conn) Write(v interface{}) error {
	chunk, err := c.enc.Encode(v)
	if err != nil {
		return err
	}
	return c.faith.Write(chunk)
}

func (c *Conn) Read(v interface{}) error {
	chunk, err := c.faith.Read()
	if err != nil {
		return err
	}
	return c.dec.Decode(chunk, v)
}

func (c *Conn) Close() error {
	return c.faith.Close()
}
