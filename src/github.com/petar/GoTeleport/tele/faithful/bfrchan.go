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
)

// BufferRead holds the return values of a call to Buffer.Read
type BufferRead struct {
	Payload interface{}
	SeqNo   SeqNo
	Err     error
}

func NewBufferReadChan(bfr *Buffer) <-chan *BufferRead {
	ch := make(chan *BufferRead)
	go func() {
		defer close(ch)
		for {
			var br BufferRead
			br.Payload, br.SeqNo, br.Err = bfr.Read()
			ch <- &br
			if br.Err == io.ErrUnexpectedEOF {
				return
			}
		}
	}()
	return ch
}
