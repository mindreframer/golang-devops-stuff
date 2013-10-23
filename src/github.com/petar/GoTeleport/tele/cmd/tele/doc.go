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

//  Package teleport implements a command-line tool for utilizing teleport transport between legacy clients and servers.
package main

import (
	"fmt"
	"os"
)

// TODO: Add limit on concurrent connections.

const help = `
Teleport Transport Tool
Part of The Go Circuit Project 2013, http://gocircuit.org
______________________________________________________________________________________
Operating diagram:

                                     client/input
                                           |
        +---------------+                  | +-------------+
        | USER's CLIENT +---- localhost -->• | TELE CLIENT +-----+
        +---------------+                    +-------------+     |
                                                                 ≈
                                                                 |
       CLIENT-SIDE                                               |
  ·····················································  UNRELIABLE NETWORK  ·····
       SERVER-SIDE                                               |
                                                                 |
                                                                 ≈
        +---------------+                    +-------------+     |
        | USER's SERVER | •<-- localhost ----+ TELE SERVER | •<--+
        +---------------+ |                  +-------------+ |
                          |                                  |
      	          server/input                server+client/output

______________________________________________________________________________________
TELEPORT SERVER:

tele -server -in=input_addr [-out=output_addr]

In server regime, the teleport tool will accept TELEPORT connections incoming
to output_addr and forward/proxy them to the TCP server listening on input_addr.
If output_addr is not specified, the tool will use an available port and print it.

______________________________________________________________________________________
TELEPORT CLIENT:

tele -client [-in=input_address] -out=output_addr

In client regime, the teleport tool will accept TCP connections incoming
to input_addr and forward/proxy them to the TELEPORT server listening on output_addr.
If input_addr is not specified, the tool will use an available port and print it.

`

/*
______________________________________________________________________________________
COMMON OPTIONS:

-max=M  Limit the number of concurrently open connections to M.
*/

func usage() {
	fatalf(help)
}

func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
