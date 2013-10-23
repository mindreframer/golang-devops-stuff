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

package tele

import (
	"github.com/petar/GoTeleport/tele/blend"
	"github.com/petar/GoTeleport/tele/chain"
	"github.com/petar/GoTeleport/tele/codec"
	"github.com/petar/GoTeleport/tele/faithful"
	"github.com/petar/GoTeleport/tele/tcp"
	"github.com/petar/GoTeleport/tele/trace"
)

func NewChunkOverTCP() *blend.Transport {
	f := trace.NewFrame()
	// Carrier
	x0 := tcp.Transport
	// Chain
	x1 := chain.NewTransport(f.Refine("chain"), x0)
	// Faithful
	x2 := faithful.NewTransport(f.Refine("faithful"), x1)
	// Codec
	x3 := codec.NewTransport(x2, codec.ChunkCodec{})
	// Blend
	return blend.NewTransport(f.Refine("blend"), x3)
}

func NewStructOverTCP() *blend.Transport {
	f := trace.NewFrame()
	// Carrier
	x0 := tcp.Transport
	// Chain
	x1 := chain.NewTransport(f.Refine("chain"), x0)
	// Faithful
	x2 := faithful.NewTransport(f.Refine("faithful"), x1)
	// Codec
	x3 := codec.NewTransport(x2, codec.GobCodec{})
	// Blend
	return blend.NewTransport(f.Refine("blend"), x3)
}
