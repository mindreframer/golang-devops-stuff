package gor

import (
	"io"
	"log"
	"net"
	"sync"
	"testing"
)

func TestTCPInput(t *testing.T) {
	wg := new(sync.WaitGroup)
	quit := make(chan int)

	input := NewTCPInput("127.0.0.1:50001")
	output := NewTestOutput(func(data []byte) {
		wg.Done()
	})

	Plugins.Inputs = []io.Reader{input}
	Plugins.Outputs = []io.Writer{output}

	go Start(quit)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		sendTCP("127.0.0.1:50001", []byte("GET / HTTP/1.1\r\n\r\n"))
	}

	wg.Wait()

	close(quit)
}

func sendTCP(addr string, data []byte) {
	tcpAddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		log.Fatal(err)
	}

	conn, err := net.DialTCP("tcp", nil, tcpAddr)
	if err != nil {
		log.Fatal(err)
	}

	conn.Write(data)
}
