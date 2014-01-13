package server_test

import (
	"net"
)

func ErrorDialingUnix(socketPath string) func() error {
	return func() error {
		conn, err := net.Dial("unix", socketPath)
		if err == nil {
			conn.Close()
		}

		return err
	}
}

func uint64ptr(n uint64) *uint64 {
	return &n
}
