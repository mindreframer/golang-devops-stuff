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

func validateRecordValue(t *testing.T, r *record, ordinal int, expected string) {
	val := r.getValue(ordinal)
	if val != expected {
		t.Errorf("values do not match expected:" + expected + " but got: " + val)
	}
}

func validateRecordValuesCount(t *testing.T, r *record, expected int) {
	valuesCount := len(r.values)
	if valuesCount != expected {
		t.Errorf("values count do not match expected:%d but got:%d", expected, valuesCount)
	}
}

func TestRecord1(t *testing.T) {
	r := newRecord(0, 0)
	validateRecordValue(t, r, 0, "0")
	r.setValue(0, "val0")
	validateRecordValue(t, r, 0, "val0")
	validateRecordValuesCount(t, r, 1)
	//
	r.setValue(1, "val1")
	validateRecordValue(t, r, 0, "val0")
	validateRecordValue(t, r, 1, "val1")
	validateRecordValuesCount(t, r, 2)
}

func TestRecord2(t *testing.T) {
	r := newRecord(5, 0)
	validateRecordValue(t, r, 0, "0")
	validateRecordValuesCount(t, r, 5)
	r.setValue(0, "val0")
	validateRecordValue(t, r, 0, "val0")
	validateRecordValuesCount(t, r, 5)
	//
	r.setValue(4, "val4")
	validateRecordValue(t, r, 0, "val0")
	validateRecordValue(t, r, 4, "val4")
	validateRecordValuesCount(t, r, 5)
	//
	r.setValue(100, "val100")
	validateRecordValue(t, r, 100, "val100")
	validateRecordValue(t, r, 99, "")
	validateRecordValue(t, r, 0, "val0")
	validateRecordValue(t, r, 4, "val4")
	validateRecordValuesCount(t, r, 101)
}
