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

package main

import (
	"flag"
	"math/rand"
	"time"

	//_ "circuit/kit/debug/ctrlc"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var (
	flagServer = flag.Bool("server", false, "Run in server regime")
	flagClient = flag.Bool("client", false, "Run in client regime")
	flagIn     = flag.String("in", "", "Input address")
	flagOut    = flag.String("out", "", "Output address")
	//flagMax    = flag.Int("max", 100, "Maximum number of concurrent connections")
)

func main() {
	flag.Parse()
	if *flagClient {
		NewClient(*flagIn, *flagOut)
	} else if *flagServer {
		NewServer(*flagIn, *flagOut)
	} else {
		usage()
	}
	<-(chan int)(nil)
}
