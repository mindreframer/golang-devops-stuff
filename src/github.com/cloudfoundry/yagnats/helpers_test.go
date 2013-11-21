package yagnats

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"sync"
	"time"
)

type FakeConnectionProvider struct {
	ReadBuffer  string
	WriteBuffer []byte
}

func (c *FakeConnectionProvider) ProvideConnection() (*Connection, error) {
	connection := NewConnection("", "", "")

	connection.conn = &fakeConn{
		ReadBuffer:  bytes.NewBuffer([]byte(c.ReadBuffer)),
		WriteBuffer: bytes.NewBuffer(c.WriteBuffer),
	}

	go connection.receivePackets()

	return connection, nil
}

type DisconnectingConnectionProvider struct {
	ReadBuffers []string
	WriteBuffer []byte
}

func (c *DisconnectingConnectionProvider) ProvideConnection() (*Connection, error) {
	if len(c.ReadBuffers) == 0 {
		return nil, errors.New("no more connections")
	}

	buf := c.ReadBuffers[0]
	c.ReadBuffers = c.ReadBuffers[1:]

	connection := NewConnection("", "", "")

	connection.conn = &fakeConn{
		ReadBuffer:  bytes.NewBuffer([]byte(buf)),
		WriteBuffer: bytes.NewBuffer(c.WriteBuffer),
	}

	go connection.receivePackets()

	return connection, nil
}

func startNats(port int) *exec.Cmd {
	cmd := exec.Command("gnatsd", "-p", strconv.Itoa(port), "--user", "nats", "--pass", "nats")
	err := cmd.Start()
	if err != nil {
		fmt.Printf("NATS failed to start: %v\n", err)
	}
	err = waitUntilNatsUp(port)
	if err != nil {
		panic("Cannot connect to NATS")
	}
	return cmd
}

func stopCmd(cmd *exec.Cmd) {
	cmd.Process.Kill()
	cmd.Wait()
}

func waitUntilNatsUp(port int) error {
	maxWait := 10
	for i := 0; i < maxWait; i++ {
		time.Sleep(500 * time.Millisecond)
		_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err == nil {
			return nil
		}
	}
	return errors.New("Waited too long for NATS to start")
}

func waitUntilNatsDown(port int) error {
	maxWait := 10
	for i := 0; i < maxWait; i++ {
		time.Sleep(500 * time.Millisecond)
		_, err := net.Dial("tcp", fmt.Sprintf("127.0.0.1:%d", port))
		if err != nil {
			return nil
		}
	}
	return errors.New("Waited too long for NATS to stop")
}

type fakeConn struct {
	ReadBuffer  *bytes.Buffer
	WriteBuffer *bytes.Buffer
	WriteChan   chan []byte
	Closed      bool

	sync.RWMutex
}

func (f *fakeConn) Read(b []byte) (n int, err error) {
	f.RLock()
	defer f.RUnlock()

	if f.Closed {
		return 0, errors.New("buffer closed")
	}

	return f.ReadBuffer.Read(b)
}

func (f *fakeConn) Write(b []byte) (n int, err error) {
	f.Lock()
	defer f.Unlock()

	if f.Closed {
		return 0, errors.New("buffer closed")
	}

	if f.WriteChan != nil {
		f.WriteChan <- b
	}

	return f.WriteBuffer.Write(b)
}

func (f *fakeConn) Close() error {
	f.Lock()
	defer f.Unlock()

	f.Closed = true
	return nil
}

func (f *fakeConn) SetDeadline(time.Time) error {
	return nil
}

func (f *fakeConn) SetReadDeadline(time.Time) error {
	return nil
}

func (f *fakeConn) SetWriteDeadline(time.Time) error {
	return nil
}

func (f *fakeConn) LocalAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:4222")
	return addr
}

func (f *fakeConn) RemoteAddr() net.Addr {
	addr, _ := net.ResolveTCPAddr("tcp", "0.0.0.0:65525")
	return addr
}
