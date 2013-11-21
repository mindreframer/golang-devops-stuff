package gor

import (
	raw "github.com/buger/gor/raw_socket_listener"
	"log"
	"net"
)

type RAWInput struct {
	data    chan []byte
	address string
}

func NewRAWInput(address string) (i *RAWInput) {
	i = new(RAWInput)
	i.data = make(chan []byte)
	i.address = address

	go i.listen(address)

	return
}

func (i *RAWInput) Read(data []byte) (int, error) {
	buf := <-i.data
	copy(data, buf)

	return len(buf), nil
}

func (i *RAWInput) listen(address string) {
	host, port, err := net.SplitHostPort(address)

	if err != nil {
		log.Fatal("input-raw: error while parsing address", err)
	}

	listener := raw.NewListener(host, port)

	for {
		// Receiving TCPMessage object
		m := listener.Receive()

		i.data <- m.Bytes()
	}
}

func (i *RAWInput) String() string {
	return "RAW Socket input: " + i.address
}
