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

import "strconv"

// link
type link struct {
	pubsub *pubsub
	tg     *tag
}

func (this *link) clear() {
	this.pubsub = nil
	this.tg = nil
}

// record
type record struct {
	values []string
	links  []link
}

// record factory
func newRecord(columns int, id int) *record {
	rec := record{
		values: make([]string, columns, columns),
	}
	rec.setValue(0, strconv.Itoa(id))
	return &rec
}

func (this *record) free() {
	this.values = nil
	this.links = nil
}

// Returns record index in a table.
func (r *record) id() int {
	id, err := strconv.Atoi(r.values[0])
	if err != nil {
		panic("record id can not be 0")
	}
	return id
}

// Returns record index in a table as string.
func (r *record) idAsString() string {
	return r.values[0]
}

// Returns value based on column ordinal.
// Empty string is returned for invalid ordinal.
func (this *record) getValue(ordinal int) string {
	if len(this.values) > ordinal {
		return this.values[ordinal]
	}
	return ""
}

// Sets value based on column ordinal.
// Automatically adjusts the record if ordinal is invalid.
func (this *record) setValue(ordinal int, val string) {
	l := len(this.values)
	if l <= ordinal {
		delta := ordinal - l + 1
		temp := make([]string, delta)
		this.values = append(this.values, temp...)
	}
	this.values[ordinal] = val
}

// addSubscription adds subscription to the record.
func (this *record) addSubscription(sub *subscription) {
	pubsb := &this.links[0].pubsub
	if *pubsb == nil {
		*pubsb = new(pubsub)
	}
	pubsb.add(sub)
}
