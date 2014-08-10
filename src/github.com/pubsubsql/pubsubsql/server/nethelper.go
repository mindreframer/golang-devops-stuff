/* Copyright (C) 2014 CompleteDB LLC.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the Apache License Version 2.0 http://www.apache.org/licenses.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
 *
 */

package server

import (
	"errors"
	"net"
	"time"
)

// message reader
type netHelper struct {
	conn  net.Conn
	bytes []byte
}

func newnetHelper(conn net.Conn, bufferSize int) *netHelper {
	var ret netHelper
	ret.set(conn, bufferSize)
	return &ret
}

func (this *netHelper) set(conn net.Conn, bufferSize int) {
	this.conn = conn
	this.bytes = make([]byte, bufferSize, bufferSize)
}

func (this *netHelper) close() {
	if this.conn != nil {
		this.conn.Close()
		this.conn = nil
	}
}

func (this *netHelper) valid() bool {
	return this.conn != nil
}

func (this *netHelper) writeMessage(bytes []byte) error {
	leftToWrite := len(bytes)
	for {
		written, err := this.conn.Write(bytes)
		if err != nil {
			return err
		}
		leftToWrite -= written
		if leftToWrite == 0 {
			break
		}
		bytes = bytes[written:]
	}
	return nil
}

func (this *netHelper) writeHeaderAndMessage(requestId uint32, bytes []byte) error {
	err := this.writeMessage(newNetHeader(uint32(len(bytes)), requestId).getBytes())
	if err != nil {
		return err
	}
	return this.writeMessage(bytes)
}

func (this *netHelper) readMessageTimeout(milliseconds int64) (*netHeader, []byte, error, bool) {
	this.conn.SetReadDeadline(time.Now().Add(time.Duration(milliseconds) * time.Millisecond))
	header, bytes, err := this.readMessage()
	timedout := false
	if neterr, ok := err.(net.Error); ok && neterr.Timeout() {
		timedout = true
		err = nil
	}
	return header, bytes, err, timedout
}

func (this *netHelper) readMessage() (*netHeader, []byte, error) {
	// header
	read, err := this.conn.Read(this.bytes[0:_HEADER_SIZE])
	if err != nil {
		return nil, nil, err
	}
	if read < _HEADER_SIZE {
		err = errors.New("Failed to read header.")
		return nil, nil, err
	}
	var header netHeader
	header.readFrom(this.bytes)
	// prepare buffer
	if len(this.bytes) < int(header.MessageSize) {
		this.bytes = make([]byte, header.MessageSize, header.MessageSize)
	}
	// message
	bytes := this.bytes[:header.MessageSize]
	left := len(bytes)
	message := bytes
	read = 0
	for left > 0 {
		bytes = bytes[read:]
		read, err = this.conn.Read(bytes)
		if err != nil {
			return nil, nil, err
		}
		left -= read
	}
	return &header, message, nil
}
