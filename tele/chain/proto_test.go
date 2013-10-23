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
	"bytes"
	"reflect"
	"testing"
)

var (
	testDialMsg    = &msgDial{ID: 7, SeqNo: 1}
	testPayloadMsg = &msgPayload{Payload: []byte{0x7, 0x2, 0x3}}
)

func TestProtoDial(t *testing.T) {
	msg := testDialMsg
	var u bytes.Buffer
	if err := msg.Write(&u); err != nil {
		t.Fatalf("dial write (%s)", err)
	}
	dial, err := readMsgDial(&u)
	if err != nil {
		t.Fatalf("dial read (%s)", err)
	}
	if !reflect.DeepEqual(dial, msg) {
		t.Fatalf("expected %#v, got %#v", msg, dial)
	}
}

func TestProtoPayload(t *testing.T) {
	msg := testPayloadMsg
	var u bytes.Buffer
	if err := msg.Write(&u); err != nil {
		t.Fatalf("payload write (%s)", err)
	}
	payload, err := readMsgPayload(&u)
	if err != nil {
		t.Fatalf("payload read (%s)", err)
	}
	if !reflect.DeepEqual(payload, msg) {
		t.Fatalf("expected %#v, got %#v", msg, payload)
	}
}
