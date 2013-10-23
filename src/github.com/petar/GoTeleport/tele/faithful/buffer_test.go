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

package faithful

import (
	"io"
	"testing"

	"github.com/petar/GoTeleport/tele/trace"
)

type testSeqNo SeqNo

func TestBuffer(t *testing.T) {
	bfr := NewBuffer(trace.NewFrame("TestBuffer"), 2)
	bfr.Write(testSeqNo(0))
	bfr.Write(testSeqNo(1))
	bfr.Remove(1)
	bfr.Write(testSeqNo(2))
	bfr.Seek(SeqNo(1))
	// Read 1
	chunk, seqno, err := bfr.Read()
	if err != nil {
		t.Fatalf("read (%s) or bad seqno", err)
	}
	if chunk != testSeqNo(1) || seqno != SeqNo(1) {
		t.Fatalf("chunk=%d seqno=%d; expecting %d", chunk, seqno, 1)
	}
	// Read 2
	chunk, seqno, err = bfr.Read()
	if err != nil {
		t.Fatalf("read (%s)", err)
	}
	if chunk != testSeqNo(2) || seqno != SeqNo(2) {
		t.Fatalf("chunk=%d seqno=%d; expecting %d", chunk, seqno, 2)
	}
	bfr.Remove(2)
	bfr.Close()
	if _, _, err := bfr.Read(); err != io.EOF {
		t.Fatalf("u (%s)", err)
	}
}
