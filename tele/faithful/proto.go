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
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"

	"github.com/petar/GoTeleport/tele/chain"
)

// MsgKind determines the type of packet following
type MsgKind byte

const (
	CHUNK = MsgKind(iota)
	SYNC
	ACK
)

// Sequence number of a chunk sent over a HiFi chunk.Conn
type SeqNo int64

const MaxSeqNoVarintLen = binary.MaxVarintLen64

// Chunk is a message containing a chunk of user data.
type Chunk struct {
	seqno SeqNo
	chunk []byte
}

type encoder interface {
	Encode() ([]byte, error)
}

func (x *Chunk) String() string {
	return fmt.Sprintf("Chunk(SeqNo=%d, Len=%d)", x.seqno, len(x.chunk))
}

func (fh *Chunk) Encode() ([]byte, error) {
	var w bytes.Buffer
	w.WriteByte(byte(CHUNK))
	q := make([]byte, MaxSeqNoVarintLen)
	n := binary.PutVarint(q, int64(fh.seqno))
	w.Write(q[:n])
	// Writing the chunk's length is not necessary, since blobs are self-delimited.
	w.Write(fh.chunk)
	return w.Bytes(), nil
}

// Sync messages are sent by the receiver endpoint of the half-connection to request a retransmit.
type Sync struct {
	NAckd SeqNo
}

func (x *Sync) String() string {
	return fmt.Sprintf("Sync(NAckd=%d)", x.NAckd)
}

func (fh *Sync) Encode() ([]byte, error) {
	var w bytes.Buffer
	w.WriteByte(byte(SYNC))
	q := make([]byte, MaxSeqNoVarintLen)
	n := binary.PutVarint(q, int64(fh.NAckd))
	w.Write(q[:n])
	return w.Bytes(), nil
}

// Ack messages are sent by the receiver endpoint of the half-connection to announce what they have received.
type Ack struct {
	NAckd SeqNo
}

func (x *Ack) String() string {
	return fmt.Sprintf("Ack(NAckd=%d)", x.NAckd)
}

func (fh *Ack) Encode() ([]byte, error) {
	var w bytes.Buffer
	w.WriteByte(byte(ACK))
	q := make([]byte, MaxSeqNoVarintLen)
	n := binary.PutVarint(q, int64(fh.NAckd))
	w.Write(q[:n])
	return w.Bytes(), nil
}

// decodeMsg decodes either a *Chunk, *Sync or *Ack object from the byte array.
func decodeMsg(q []byte) (interface{}, error) {
	r := bytes.NewReader(q)
	// MsgKind
	msgkind, err := r.ReadByte()
	if err != nil {
		return nil, err
	}
	msgKind := MsgKind(msgkind)
	// Switch
	switch msgKind {
	case CHUNK:
		msg := &Chunk{}
		// SeqNo
		seqno, err := binary.ReadVarint(r)
		if err != nil {
			return nil, err
		}
		msg.seqno = SeqNo(seqno)
		// Chunk
		msg.chunk, err = ioutil.ReadAll(r)
		if err != nil {
			panic("u")
		}
		return msg, nil

	case SYNC:
		msg := &Sync{}
		nackd, err := binary.ReadVarint(r)
		if err != nil {
			return nil, err
		}
		msg.NAckd = SeqNo(nackd)
		return msg, nil

	case ACK:
		msg := &Ack{}
		nackd, err := binary.ReadVarint(r)
		if err != nil {
			return nil, err
		}
		msg.NAckd = SeqNo(nackd)
		return msg, nil
	}
	return nil, chain.ErrMisbehave
}
