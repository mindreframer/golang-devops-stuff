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
	"errors"
	"fmt"
)

var (
	// ErrMisbehave indicates that the remote endpoint is not behaving to protocol.
	ErrMisbehave = errors.New("misbehave")

	errDup = errors.New("duplicate")
)

// IsStitch returns true if err is a stitching error.
func IsStitch(err error) *ConnWriter {
	if err == nil {
		return nil
	}
	es, ok := err.(*ErrStitch)
	if !ok {
		return nil
	}
	return es.Writer
}

type ErrStitch struct {
	SeqNo  SeqNo
	Writer *ConnWriter
}

func (es *ErrStitch) Error() string {
	return fmt.Sprintf("stitch #%d", es.SeqNo)
}
