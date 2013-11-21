package gor

import (
	"io"
	"log"
	"net"
	"sync"
	"testing"
)

func TestTCPOutput(t *testing.T) {
	wg := new(sync.WaitGroup)
	quit := make(chan int)

	input := NewTestInput()
	output := NewTCPOutput(":50002")

	startTCP(":50002", func(data []byte) {
		wg.Done()
	})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}

	go Start(quit)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		input.EmitGET()
	}

	wg.Wait()

	close(quit)
}

func startTCP(addr string, cb func([]byte)) {
	listener, err := net.Listen("tcp", addr)

	if err != nil {
		log.Fatal("Can't start:", err)
	}

	go func() {
		for {
			conn, _ := listener.Accept()

			var read = true
			var response []byte
			var buf []byte

			buf = make([]byte, 4094)

			for read {
				n, err := conn.Read(buf)

				switch err {
				case io.EOF:
					read = false
				case nil:
					response = append(response, buf[:n]...)
					if n < 4096 {
						read = false
					}
				default:
					read = false
				}
			}

			cb(response)

			conn.Close()
		}
	}()
}
