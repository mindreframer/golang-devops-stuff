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

package trace

import (
	"fmt"
)

type Op int

const (
	READ = Op(iota)
	WRITE
)

func PrintOp(err error, proto string, op Op, msg fmt.Stringer) string {
	var t string
	switch op {
	case READ:
		t = "READ "
	case WRITE:
		t = "WROTE"
	default:
		t = "UKNWN"
	}
	var e string
	if err != nil {
		e = fmt.Sprintf("ERROR(%s)", err)
	}
	return fmt.Sprintf("EVE %5s %5s %s %s", proto, t, msg, e)
}

func DeferPrintOp(frame Frame, err *error, proto string, op Op, msg fmt.Stringer) {
	frame.Println(PrintOp(*err, proto, op, msg))
}
