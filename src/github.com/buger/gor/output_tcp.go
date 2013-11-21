package gor

import (
	"fmt"
	"io"
	"log"
	"net"
	"strconv"
	"strings"
)

type TCPOutput struct {
	address string
	limit   int
}

func NewTCPOutput(options string) io.Writer {
	o := new(TCPOutput)

	optionsArr := strings.Split(options, "|")
	o.address = optionsArr[0]

	if len(optionsArr) > 1 {
		o.limit, _ = strconv.Atoi(optionsArr[1])
	}

	if o.limit > 0 {
		return NewLimiter(o, o.limit)
	} else {
		return o
	}
}

func (o *TCPOutput) Write(data []byte) (n int, err error) {
	conn, err := o.connect(o.address)
	defer conn.Close()

	if err != nil {
		n, err = conn.Write(data)
	}

	return
}

func (o *TCPOutput) connect(address string) (conn net.Conn, err error) {
	conn, err = net.Dial("tcp", address)

	if err != nil {
		log.Println("Connection error ", err, o.address)
	}

	return
}

func (o *TCPOutput) String() string {
	return fmt.Sprintf("TCP output %s, limit: %d", o.address, o.limit)
}
