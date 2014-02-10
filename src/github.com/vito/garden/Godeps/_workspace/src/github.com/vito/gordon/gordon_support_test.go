package gordon_test

import (
	"bytes"
	"errors"
	. "github.com/vito/gordon"
	. "github.com/vito/gordon/test_helpers"

	"github.com/vito/gordon/connection"
)

type FailingConnectionProvider struct{}

func (c *FailingConnectionProvider) ProvideConnection() (*connection.Connection, error) {
	return nil, errors.New("nope!")
}

type FakeConnectionProvider struct {
	connection *connection.Connection
}

func NewFakeConnectionProvider(readBuffer, writeBuffer *bytes.Buffer) *FakeConnectionProvider {
	return &FakeConnectionProvider{
		connection: connection.New(
			&FakeConn{
				ReadBuffer:  readBuffer,
				WriteBuffer: writeBuffer,
			},
		),
	}
}

func (c *FakeConnectionProvider) ProvideConnection() (*connection.Connection, error) {
	return c.connection, nil
}

type ManyConnectionProvider struct {
	ConnectionProviders []ConnectionProvider
}

func (c *ManyConnectionProvider) ProvideConnection() (*connection.Connection, error) {
	if len(c.ConnectionProviders) == 0 {
		return nil, errors.New("no more connections")
	}

	cp := c.ConnectionProviders[0]
	c.ConnectionProviders = c.ConnectionProviders[1:]

	return cp.ProvideConnection()
}
