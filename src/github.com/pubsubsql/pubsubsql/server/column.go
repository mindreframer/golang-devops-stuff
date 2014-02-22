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

type columnType int8

// column types
const (
	columnTypeNormal columnType = iota // non indexed column
	columnTypeId                       // id column
	columnTypeKey                      // key column
	columnTypeTag                      // tag column
)

// column
type column struct {
	name    string
	ordinal int
	typ     columnType
	//
	tagmap   tagMap
	tagIndex int
}

// column factory
func newColumn(name string, ordinal int) *column {
	col := &column{
		name:     name,
		ordinal:  ordinal,
		typ:      columnTypeNormal,
		tagIndex: -1,
	}
	if col.ordinal == 0 {
		col.typ = columnTypeId
	}
	return col
}

func (this *column) isKey() bool {
	return this.typ == columnTypeKey
}

func (this *column) isTag() bool {
	return this.typ == columnTypeTag
}

func (this *column) isIndexed() bool {
	return this.typ != columnTypeNormal
}

// Makes column to be tags container.
func (this *column) makeTags(tagIndex int) {
	this.typ = columnTypeTag
	this.tagmap.init()
	this.tagIndex = tagIndex
}

// Determines if value is present for a given key
func (this *column) keyContainsValue(key string) bool {
	return this.tagmap.containsTag(key)
}
