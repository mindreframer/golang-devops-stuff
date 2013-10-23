/*
Copyright 2013 Google Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"bytes"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"time"
)

const bufSize = 1024

var (
	listenAddr   = flag.String("listen", ":2222", "local listen address")
	httpAddr     = flag.String("http", "127.0.0.1:8888", "remote tunnel server")
	destAddr     = flag.String("dest", "127.0.0.1:22", "tunnel destination")
	tickInterval = flag.Int("tick", 250, "update interval (msec)")
)

// take a reader, and turn it into a channel of bufSize chunks of []byte
func makeReadChan(r io.Reader, bufSize int) chan []byte {
	read := make(chan []byte)
	go func() {
		for {
			b := make([]byte, bufSize)
			n, err := r.Read(b)
			if err != nil {
				return
			}
			if n > 0 {
				read <- b[0:n]
			}
		}
	}()
	return read
}

func main() {
	flag.Parse()
	log.SetPrefix("httptun.c: ")

	listener, err := net.Listen("tcp", *listenAddr)
	if err != nil {
		panic(err)
	}
	log.Println("listen", *listenAddr)

	conn, err := listener.Accept()
	if err != nil {
		panic(err)
	}
	log.Println("accept conn", "localAddr.", conn.LocalAddr(), "remoteAddr.", conn.RemoteAddr())

	buf := new(bytes.Buffer)

	// initiate new session and read key
	log.Println("Attempting connect HttpTun Server.", *httpAddr, "for dest.", *destAddr)
	buf.Write([]byte(*destAddr))
	resp, err := http.Post(
		"http://"+*httpAddr+"/create",
		"text/plain",
		buf)
	if err != nil {
		panic(err)
	}
	key, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()

	log.Println("ResponseWriterected, key", key)

	// ticker to set a rate at which to hit the server
	tick := time.NewTicker(time.Duration(int64(*tickInterval)) * time.Millisecond)
	read := makeReadChan(conn, bufSize)
	buf.Reset()
	for {
		select {
		case <-tick.C:
			// write buf to new http request
			req := bytes.NewBuffer(key)
			buf.WriteTo(req)
			resp, err := http.Post(
				"http://"+*httpAddr+"/ping",
				"application/octet-stream",
				req)
			if err != nil {
				log.Println(err.Error())
				continue
			}
			// write http response response to conn
			io.Copy(conn, resp.Body)
			resp.Body.Close()
		case b := <-read:
			buf.Write(b)
		}
	}
}
