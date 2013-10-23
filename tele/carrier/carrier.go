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

// Package carrier defines the interface of the transport underlying the chain transport layer.
package carrier

import (
	"errors"
	"net"
)

// Transport represents a transport underlying the chain transport layer.
// Transport is a reliable, connection-oriented transport akin to TCP.
type Transport interface {

	// Listen returns a new listener that listens on the given opaque address.
	Listen(net.Addr) (net.Listener, error)

	// Dial tries to establish a connection with the addressed remote endpoint.
	// Dial must distinguish between temporary and permanent obstructions to dialing the destination.
	// An ErrPerm error should be returned if addressed entity is permanently (from a logical standpoint) dead.
	// All other errors are considered temporary obstructions.
	Dial(net.Addr) (net.Conn, error)
}

var ErrPerm = errors.New("remote permanently gone")
