package gor

import (
	"io"
	"log"
	"net"
)

// Can be tested using nc tool:
//    echo "asdad" | nc 127.0.0.1 27017
//
type TCPInput struct {
	data    chan []byte
	address string
}

func NewTCPInput(address string) (i *TCPInput) {
	i = new(TCPInput)
	i.data = make(chan []byte)
	i.address = address

	i.listen(address)

	return
}

func (i *TCPInput) Read(data []byte) (int, error) {
	buf := <-i.data
	copy(data, buf)

	return len(buf), nil
}

func (i *TCPInput) listen(address string) {
	listener, err := net.Listen("tcp", address)

	if err != nil {
		log.Fatal("Can't start:", err)
	}

	go func() {
		for {
			conn, err := listener.Accept()

			if err != nil {
				log.Println("Error while Accept()", err)
				continue
			}

			go i.handleConnection(conn)
		}
	}()
}

func (i *TCPInput) handleConnection(conn net.Conn) {
	defer conn.Close()

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

	i.data <- response
}

func (i *TCPInput) String() string {
	return "TCP input: " + i.address
}
