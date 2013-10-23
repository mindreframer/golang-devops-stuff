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

package blend

import (
	"encoding/gob"
)

type (
	ConnID uint32
	SeqNo  uint32
)

type PayloadMsg struct {
	SeqNo   SeqNo
	Payload interface{} // User-supplied type that can be coded by the underlying codec
}

type CloseMsg struct{}

type Msg struct {
	ConnID ConnID
	Demux  interface{} // PayloadMsg or CloseMsg
}

func init() {
	gob.Register(&PayloadMsg{})
	gob.Register(&CloseMsg{})
}
