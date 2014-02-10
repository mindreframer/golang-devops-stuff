package server_test

import (
	"net"
)

func ErrorDialing(network, addr string) func() error {
	return func() error {
		conn, err := net.Dial(network, addr)
		if err == nil {
			conn.Close()
		}

		return err
	}
}

func uint64ptr(n uint64) *uint64 {
	return &n
}
