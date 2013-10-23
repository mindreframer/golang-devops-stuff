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
	"reflect"
	"testing"
)

const gobN = 7

type testBlob struct {
	A int
	B int
}

func TestGobCodec(t *testing.T) {
	enc := (GobCodec{}).NewEncoder()
	dec := (GobCodec{}).NewDecoder()
	u := &testBlob{A: 1, B: 5}
	for i := 0; i < gobN; i++ {
		chunk, err := enc.Encode(u)
		if err != nil {
			t.Fatalf("encode (%s)", err)
		}
		v := &testBlob{}
		if err = dec.Decode(chunk, v); err != nil {
			t.Fatalf("decode (%s)", err)
		}
		if !reflect.DeepEqual(v, u) {
			t.Fatalf("not equal")
		}
	}
}
