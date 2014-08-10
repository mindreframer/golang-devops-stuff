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

package server

type removeTagReturn int8

const (
	removeTagLast  removeTagReturn = iota // indicates that last element was removed
	removeTagSlide                        // indicates that slide has happened external pointer need to be updated
	removeTagNormal
)

// tag implemented as doubly linked list.
type tag struct {
	prev *tag
	next *tag
	idx  int // idx index into table.records
}

// Adds tag as next element after the head.
// Returns added tag.
func addTag(head *tag, idx int) *tag {
	t := &tag{
		idx: idx,
	}
	if head != nil {
		head.next, t.next, t.prev = t, head.next, head
	}
	if t.next != nil {
		t.next.prev = t
	}
	return t
}

// Removes tag from the list.
// Returns true if last element was removed.
func removeTag(t *tag) removeTagReturn {
	ret := removeTagNormal
	freeMe := t
	// handle head case
	if t.prev == nil {
		if t.next == nil {
			// last element, let caller(columns tag map) handle the rest
			return removeTagLast
		}
		// slide and remove
		ret = removeTagSlide
		freeMe = t.next
		t.idx, t.next = freeMe.idx, freeMe.next
		if t.next != nil {
			t.next.prev = t
		}
	} else {
		t.prev.next = t.next
		if t.next != nil {
			t.next.prev = t.prev
		}
	}
	// let GC know that we need to go....
	freeMe.prev = nil
	freeMe.next = nil
	return ret
}
