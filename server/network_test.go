/* Copyright (C) 2013 CompleteDB LLC.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as
 * published by the Free Software Foundation, either version 3 of the
 * License, or (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU Affero General Public License for more details.
 *
 * You should have received a copy of the GNU Affero General Public License
 * along with PubSubSQL.  If not, see <http://www.gnu.org/licenses/>.
 */

package server

import (
	"net"
	"testing"
	"time"
)

func TestNetworkStartQuit(t *testing.T) {
	debug("TestNetworkStartQuit")
	address := "localhost:54321"
	context := newNetworkContextStub()
	s := context.quit
	n := newNetwork(context)
	if !n.start(address) {
		t.Error("network.start(" + address + ") failed")
	}
	//
	n2 := newNetwork(newNetworkContextStub())
	if n2.start(address) {
		t.Error("network.start(" + address + ") should have failed")
	}
	// shutdown
	s.Quit(0)
	n.stop()
	s.Wait(time.Millisecond * 1000)
}

func TestNetworkConnections(t *testing.T) {
	debug("TestNetworkConnections")
	context := newNetworkContextStub()
	s := context.quit
	n := newNetwork(context)
	n.start("localhost:54321")
	c, err := net.Dial("tcp", "localhost:54321")
	if err != nil {
		t.Error(err)
	}
	time.Sleep(time.Millisecond * 1000)
	if n.connectionCount() != 1 {
		t.Error("Expected 1 network connection")
	}
	// shutdown
	s.Quit(0)
	n.stop()
	s.Wait(time.Millisecond * 1000)
	c.Close()
}

func validateWriteRead(t *testing.T, conn net.Conn, message string, requestId uint32) {
	rw := newnetHelper(conn, config.NET_READWRITE_BUFFER_SIZE)
	bytes := []byte(message)
	var header *netHeader
	err := rw.writeHeaderAndMessage(requestId, bytes)
	if err != nil {
		t.Error(err)
	}
	header, bytes, err = rw.readMessage()
	if err != nil {
		t.Error(err)
	}
	if header.RequestId != requestId {
		t.Error("Expected requestid", requestId, "but got", header.RequestId, " command:", message)
	}
	debug(string(bytes))
}

func validateRead(t *testing.T, conn net.Conn, requestId uint32) {
	rw := newnetHelper(conn, config.NET_READWRITE_BUFFER_SIZE)
	header, bytes, err := rw.readMessage()
	if err != nil {
		t.Error(err)
	}
	if header.RequestId != requestId {
		t.Error("Request id 0 does not match ")
	}
	debug(string(bytes))
}

func validateConnect(t *testing.T, address string) net.Conn {
	c, err := net.Dial("tcp", address)
	if err != nil {
		t.Error(err)
	}
	return c
}

func TestNetworkWriteRead(t *testing.T) {
	debug("TestNetworkReadWrite")
	context := newNetworkContextStub()
	address := "localhost:54321"
	s := context.quit
	n := newNetwork(context)
	n.start(address)
	c := validateConnect(t, address)
	// send valid message get result
	validateWriteRead(t, c, "key stocks ticker", 1)
	validateWriteRead(t, c, "bla bla bla", 2)
	validateWriteRead(t, c, "insert into stocks (ticker, bid, ask) values (IBM,123,124)", 3)
	validateWriteRead(t, c, "insert into stocks (ticker, bid, ask) values (MSFT,37,38.45)", 4)
	validateWriteRead(t, c, "select * from stocks", 5)
	validateWriteRead(t, c, "key stocks ticker", 6)
	// test pubsub
	c2 := validateConnect(t, address)
	validateWriteRead(t, c2, "subscribe * from stocks", 7)
	// on add
	validateRead(t, c2, 0)
	validateWriteRead(t, c, "insert into stocks (ticker, bid, ask) values (ORCL,37,38.45)", 8)
	// on insert
	validateRead(t, c2, 0)
	//
	if n.connectionCount() != 2 {
		t.Error("Expected 1 network connection")
	}
	// close connections
	c.Close()
	time.Sleep(time.Millisecond * 60)
	if n.connectionCount() != 1 {
		t.Error("Expected 1 network connection")
	}
	c2.Close()
	time.Sleep(time.Millisecond * 60)
	if n.connectionCount() != 0 {
		t.Error("Expected 0 network connection")
	}
	// shutdown
	s.Quit(0)
	n.stop()
	s.Wait(time.Millisecond * 500)
}

func TestNetworkBatchRead(t *testing.T) {
	context := newNetworkContextStub()
	address := "localhost:54321"
	s := context.quit
	n := newNetwork(context)
	n.start(address)
	c := validateConnect(t, address)

	prevBatchSize := config.DATA_BATCH_SIZE
	config.DATA_BATCH_SIZE = 1
	defer func() {
		config.DATA_BATCH_SIZE = prevBatchSize
	}()
	// insert
	validateWriteRead(t, c, "insert into stocks (ticker, bid) values (IBM, 120)", 1)
	validateWriteRead(t, c, "insert into stocks (ticker, bid) values (MSFT, 120)", 2)
	validateWriteRead(t, c, "insert into stocks (ticker, bid) values (GOOG, 120)", 3)
	validateWriteRead(t, c, "insert into stocks (ticker, bid) values (ORCL, 120)", 4)
	//expected another 3 messages
	validateWriteRead(t, c, "select * from stocks", 5)
	validateRead(t, c, 5)
	validateRead(t, c, 5)
	validateRead(t, c, 5)

	c.Close()
	// shutdown
	s.Quit(0)
	n.stop()
	s.Wait(time.Millisecond * 500)
}
