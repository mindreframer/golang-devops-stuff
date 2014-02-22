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
 * You should have idxeived a copy of the GNU Affero General Public License
 * along with PubSubSQL.  If not, see <http://www.gnu.org/licenses/>.
 */

package pubsubsql

import (
	"bytes"
	"strconv"
	"unicode/utf8"
)

type JSONBuilder struct {
	bytes.Buffer
	err bool
}

func networkReadyJSONBuilder() *JSONBuilder {
	builder := new(JSONBuilder)
	builder.Write(_EMPTY_HEADER)
	return builder
}

// implementation for string function was copied from go source code
var hex = "0123456789abcdef"

func (this *JSONBuilder) string(s string) int {
	len0 := this.Len()
	this.WriteByte('"')
	start := 0
	for i := 0; i < len(s); {
		if b := s[i]; b < utf8.RuneSelf {
			if 0x20 <= b && b != '\\' && b != '"' && b != '<' && b != '>' {
				i++
				continue
			}
			if start < i {
				this.WriteString(s[start:i])
			}
			switch b {
			case '\\', '"':
				this.WriteByte('\\')
				this.WriteByte(b)
			case '\n':
				this.WriteByte('\\')
				this.WriteByte('n')
			case '\r':
				this.WriteByte('\\')
				this.WriteByte('r')
			default:
				// This encodes bytes < 0x20 except for \n and \r,
				// as well as < and >. The latter are escaped because they
				// can lead to security holes when user-controlled strings
				// are rendered into JSON and served to some browsers.
				this.WriteString(`\u00`)
				this.WriteByte(hex[b>>4])
				this.WriteByte(hex[b&0xF])
			}
			i++
			start = i
			continue
		}
		c, size := utf8.DecodeRuneInString(s[i:])
		if c == utf8.RuneError && size == 1 {
			this.err = true
		}
		i += size
	}
	if start < len(s) {
		this.WriteString(s[start:])
	}
	this.WriteByte('"')
	return this.Len() - len0
}

func (this *JSONBuilder) int(i int) {
	this.WriteString(strconv.Itoa(i))
}

func (this *JSONBuilder) beginArray() {
	this.WriteByte('[')
}

func (this *JSONBuilder) beginObject() {
	this.WriteByte('\n')
	this.WriteByte('{')
}

func (this *JSONBuilder) endArray() {
	this.WriteByte(']')
}

func (this *JSONBuilder) endObject() {
	this.WriteByte('}')
}

func (this *JSONBuilder) nameSeparator() {
	this.WriteByte(':')
}

func (this *JSONBuilder) valueSeparator() {
	this.WriteByte(',')
}

func (this *JSONBuilder) objectSeparator() {
	this.WriteByte(',')
}

func (this *JSONBuilder) nameValue(name string, value string) {
	this.string(name)
	this.nameSeparator()
	this.string(value)
}

func (this *JSONBuilder) newLine() {
	this.WriteByte('\n')
}

func (this *JSONBuilder) nameIntValue(name string, val int) {
	this.string(name)
	this.nameSeparator()
	this.int(val)
}

func (this *JSONBuilder) getNetworkBytes(requestId uint32) []byte {
	bytes := this.Bytes()
	var header netHeader
	header.MessageSize = uint32(len(bytes)) - uint32(_HEADER_SIZE)
	header.RequestId = requestId
	header.writeTo(bytes)
	return bytes
}

//this function is for testing only
func (this *JSONBuilder) getBytes() []byte {
	return fromNetworkBytes(this.getNetworkBytes(0))
}

func fromNetworkBytes(bytes []byte) []byte {
	if len(bytes) > _HEADER_SIZE {
		return bytes[_HEADER_SIZE:]
	}
	return nil
}
