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
	"log"
	"net"
	"os"
)

var flagAddr = flag.String("addr", ":8787", "Address to listen to")

func main() {
	flag.Parse()
	l, err := net.Listen("tcp", *flagAddr)
	if err != nil {
		log.Printf("accept (%s)", err)
		os.Exit(1)
	}
	go loop(l)
	<-(chan int)(nil)
}

func loop(l net.Listener) {
	for {
		conn, err := l.Accept()
		if err != nil {
			log.Printf("accept (%s)", err)
			os.Exit(1)
		}
		go func() {
			defer func() {
				conn.Close()
				log.Printf("closed %s", conn.RemoteAddr())
			}()
			log.Printf("accepted %s", conn.RemoteAddr())
			for i := 0; i < 3; i++ {
				p := make([]byte, 10)
				n, _ := conn.Read(p)
				log.Printf("read from %s: buf=%s err=%v", conn.RemoteAddr(), string(p[:n]), err)
				m, err := conn.Write(p[:n])
				log.Printf("wrote to %s: buf=%s err=%v", conn.RemoteAddr(), string(p[:m]), err)
				if err != nil {
					return
				}
				conn.Write([]byte("--\n"))
			}
		}()
	}
}
