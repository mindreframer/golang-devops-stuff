package network

import (
	"net"
)

// Streaming transports like TCP and UNIX sockets can be in a half-closed state
type HalfCloseable interface {
	CloseRead() error
	CloseWrite() error
}

type fullDuplex struct {
	net.Conn
	readable, writeable bool
}

func NewSockWrap(conn net.Conn) *fullDuplex {
	return &fullDuplex{
		conn,
		true,
		true,
	}
}

func (s *fullDuplex) Close() error {
	s.readable = false
	s.writeable = false
	return s.Conn.Close()
}

func (s *fullDuplex) CloseRead() error {
	s.readable = false
	if s.writeable {
		return s.Conn.(HalfCloseable).CloseRead()
	} else {
		return s.Conn.Close()
	}
}

func (s *fullDuplex) CloseWrite() error {
	s.writeable = false
	if s.readable {
		return s.Conn.(HalfCloseable).CloseWrite()
	} else {
		return s.Conn.Close()
	}
}
