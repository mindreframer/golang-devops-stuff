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

// Codec
type Codec interface {
	NewEncoder() Encoder
	NewDecoder() Decoder
}

// Encoder
type Encoder interface {
	Encode(interface{}) ([]byte, error)
}

// Decoder
type Decoder interface {
	Decode([]byte, interface{}) error
}

// ChunkCodec
type ChunkCodec struct{}

func (ChunkCodec) NewEncoder() Encoder {
	return ChunkEncoder{}
}

func (ChunkCodec) NewDecoder() Decoder {
	return ChunkDecoder{}
}

// ChunkEncoder
type ChunkEncoder struct{}

func (ChunkEncoder) Encode(v interface{}) ([]byte, error) {
	return v.([]byte), nil
}

// ChunkDecoder
type ChunkDecoder struct{}

func (ChunkDecoder) Decode(r []byte, v interface{}) error {
	v = r
	return nil
}
