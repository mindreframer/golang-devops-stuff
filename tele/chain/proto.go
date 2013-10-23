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
	"encoding/binary"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
)

// chainID identifies a chain of underlying connections uniquely
type chainID uint64

const maxVarintChainIDLen = binary.MaxVarintLen64

func chooseChainID() chainID {
	return chainID(rand.Int63())
}

// SeqNo indexes an underlying connection by its order within a chain
type SeqNo uint32

const MaxVarintSeqNoLen = binary.MaxVarintLen32

// msgDial
type msgDial struct {
	ID    chainID
	SeqNo SeqNo // 1 = dial, 2 = redial, and so on
}

const maxMsgDialLen = maxVarintChainIDLen + MaxVarintSeqNoLen

func (x *msgDial) String() string {
	return fmt.Sprintf("Dial(ID=%x, SeqNo=%d)", x.ID, x.SeqNo)
}

func (x *msgDial) Write(w io.Writer) (err error) {
	// In order to play well with the sandbox tests, we require that Write utilizes exactly one
	// Write request to w. To do so, we first prepare the encoding and then we write it as one.
	var u bytes.Buffer
	if err = x.write(&u); err != nil {
		panic("u")
	}
	_, err = w.Write(u.Bytes())
	return err
}

func (x *msgDial) write(w io.Writer) (err error) {
	q := make([]byte, maxMsgDialLen)
	n1 := binary.PutUvarint(q, uint64(x.ID))
	n2 := binary.PutUvarint(q[n1:], uint64(x.SeqNo))
	_, err = w.Write(q[:n1+n2])
	return err
}

func readMsgDial(r io.ByteReader) (*msgDial, error) {
	id, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, err
	}
	seqno, err := binary.ReadUvarint(r)
	if err != nil {
		return nil, err
	}
	return &msgDial{ID: chainID(id), SeqNo: SeqNo(seqno)}, nil
}

// msgPayload
type msgPayload struct {
	Payload []byte
}

const MaxPayloadSize = 1e5 // 100K

func (x *msgPayload) String() string {
	return fmt.Sprintf("Payload(Len=%d)", len(x.Payload))
}

func (x *msgPayload) Write(w io.Writer) (err error) {
	// In order to play well with the sandbox tests, we require that Write utilizes exactly one
	// Write request to w. To do so, we first prepare the encoding and then we write it as one.
	var u bytes.Buffer
	if err = x.write(&u); err != nil {
		panic("u")
	}
	_, err = w.Write(u.Bytes())
	return err
}

func (x *msgPayload) write(w io.Writer) (err error) {
	if len(x.Payload) > MaxPayloadSize {
		panic("payload excess")
	}
	q := make([]byte, binary.MaxVarintLen32)
	n1 := binary.PutVarint(q, int64(len(x.Payload)))
	if _, err = w.Write(q[:n1]); err != nil {
		return err
	}
	n2, err := w.Write(x.Payload)
	if n2 != len(x.Payload) {
		return io.ErrShortWrite
	}
	return err
}

func readMsgPayload(r interface {
	io.ByteReader
	io.Reader
}) (*msgPayload, error) {

	l, err := binary.ReadVarint(r)
	if err != nil {
		return nil, err
	}
	if l > MaxPayloadSize {
		return nil, ErrMisbehave
	}
	q, err := ioutil.ReadAll(&io.LimitedReader{R: r, N: l})
	if len(q) == int(l) {
		return &msgPayload{Payload: q}, nil
	}
	if err != nil {
		return nil, err
	}
	return nil, io.ErrShortBuffer
}
