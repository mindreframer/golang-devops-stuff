package test_helpers

import (
	"bytes"
	"errors"
	"net"
	"time"
)

type FakeConn struct {
	ReadBuffer  *bytes.Buffer
	WriteBuffer *bytes.Buffer
	WriteChan   chan string
	Closed      bool
}

func (f *FakeConn) Read(b []byte) (n int, err error) {
	if f.Closed {
		return 0, errors.New("buffer closed")
	}

	return f.ReadBuffer.Read(b)
}

func (f *FakeConn) Write(b []byte) (n int, err error) {
	if f.Closed {
		return 0, errors.New("buffer closed")
	}

	if f.WriteChan != nil {
		f.WriteChan <- string(b)
	}

	return f.WriteBuffer.Write(b)
}

func (f *FakeConn) Close() error {
	f.Closed = true
	return nil
}

func (f *FakeConn) SetDeadline(time.Time) error {
	return nil
}

func (f *FakeConn) SetReadDeadline(time.Time) error {
	return nil
}

func (f *FakeConn) SetWriteDeadline(time.Time) error {
	return nil
}

func (f *FakeConn) LocalAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:4222")
	return addr
}

func (f *FakeConn) RemoteAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:65525")
	return addr
}
