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

package pubsubsql

import "testing"
import "encoding/json"

//import "fmt"

func validateBuilder(builder *JSONBuilder, t *testing.T) {
	var v interface{}
	err := json.Unmarshal(builder.getBytes(), &v)
	if err != nil {
		t.Error("failed to validate JSONBuilder:", err)
		t.Error(string(builder.getBytes()))
	}
}

func TestEmptyObject(t *testing.T) {
	builder := networkReadyJSONBuilder()
	builder.beginObject()
	builder.endObject()
	validateBuilder(builder, t)
}

func TestObject(t *testing.T) {
	builder := networkReadyJSONBuilder()
	builder.beginObject()
	builder.nameValue("status", "ok")
	builder.endObject()
	validateBuilder(builder, t)
}

func TestInt(t *testing.T) {
	builder := networkReadyJSONBuilder()
	builder.beginObject()
	builder.nameIntValue("rows", 123456)
	builder.endObject()
	validateBuilder(builder, t)
}

func TestObject2(t *testing.T) {
	builder := networkReadyJSONBuilder()
	builder.beginObject()
	builder.nameValue("status", "ok")
	builder.valueSeparator()
	builder.nameValue("somename", "somevalue")
	builder.endObject()
	validateBuilder(builder, t)
}

func TestObject3(t *testing.T) {
	builder := networkReadyJSONBuilder()
	builder.beginObject()
	builder.nameValue("status", "ok")
	builder.valueSeparator()
	builder.string("somename")
	builder.nameSeparator()
	builder.beginObject()
	builder.nameValue("status", "ok")
	builder.valueSeparator()
	builder.nameValue("somename", "somevalue")
	builder.endObject()
	builder.endObject()
	validateBuilder(builder, t)
}

func TestArray(t *testing.T) {
	builder := networkReadyJSONBuilder()
	builder.beginObject()
	builder.nameValue("status", "ok")
	builder.valueSeparator()
	builder.string("data")
	builder.nameSeparator()
	builder.beginArray()
	for i := 0; i < 10; i++ {
		if i > 0 {
			builder.valueSeparator()
		}
		builder.beginObject()
		builder.nameValue("status", "ok")
		builder.valueSeparator()
		builder.nameValue("somename", "somevalue")
		builder.endObject()
	}
	builder.endArray()
	builder.endObject()
	validateBuilder(builder, t)
}
