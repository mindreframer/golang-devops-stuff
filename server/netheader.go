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

package pubsubsql

import (
	"encoding/binary"
	"encoding/json"
)

/*
--------------------+--------------------
|   message size    |    request id     |
--------------------+--------------------
|      uint32       |      uint32       |
--------------------+--------------------
*/

type netHeader struct {
	MessageSize uint32
	RequestId   uint32
}

var _HEADER_SIZE = 8
var _EMPTY_HEADER = make([]byte, _HEADER_SIZE, _HEADER_SIZE)

func newNetHeader(messageSize uint32, requestId uint32) *netHeader {
	return &netHeader{
		MessageSize: messageSize,
		RequestId:   requestId,
	}
}

func (this *netHeader) readFrom(bytes []byte) {
	this.MessageSize = binary.BigEndian.Uint32(bytes)
	this.RequestId = binary.BigEndian.Uint32(bytes[4:])
}

func (this *netHeader) writeTo(bytes []byte) {
	binary.BigEndian.PutUint32(bytes, this.MessageSize)
	binary.BigEndian.PutUint32(bytes[4:], this.RequestId)
}

func (this *netHeader) getBytes() []byte {
	bytes := make([]byte, _HEADER_SIZE, _HEADER_SIZE)
	this.writeTo(bytes)
	return bytes
}

func (this *netHeader) String() string {
	bytes, _ := json.Marshal(this)
	return string(bytes)
}
