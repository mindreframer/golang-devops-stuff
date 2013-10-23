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

// Package tele implements Teleport Transport which can overcome network outages without affecting endpoint logic.
package tele

/*

	The Teleport Transport networking stack:

	+----------------+
	|     BLEND      | Logical connection de/multiplexing over a single underlying connection.
	+----------------+
	|     CODEC      | Per-faithful-connection, stateful encoding/decoding layer, e.g. gob/ProtoBuf/etc.
	+----------------+
	|    FAITHFUL    | Recover dropped messages from lossy connection.
	+----------------+
	|     CHAIN      | Linkup of multiple disconnect-prone carrier connections into a single lossy disconnect-less connection.
	+----------------+
	|    CARRIER     | Underlying transport, e.g. TCP/WebRTC/sandbox/etc.
	+----------------+

*/
