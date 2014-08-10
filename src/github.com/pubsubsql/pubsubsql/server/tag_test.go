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

import "testing"

func invalidNextPrev(tg *tag) bool {
	if tg.next != nil {
		if tg.next.prev != tg {
			return true
		}
		return invalidNextPrev(tg.next)
	}
	return false
}

func TestTag(t *testing.T) {
	i := 1
	head := addTag(nil, i)
	if head == nil {
		t.Errorf("expected valid tag")
	}
	// 1
	if head.idx != i {
		t.Errorf("expected %d", i)
	}
	// 1 -> 2
	i = 2
	tg := addTag(head, i)
	if tg.idx != i {
		t.Errorf("expected %d", i)
	}
	// 1 -> 5 -> 2
	i = 5
	tg = addTag(head, i)
	if tg.idx != i {
		t.Errorf("expected %d", i)
	}
	//
	if tg.prev.idx != 1 {
		t.Errorf("expected %d", 1)
	}
	//
	if tg.next.idx != 2 {
		t.Errorf("expected %d", 2)
	}
	//
	if tg.prev != head {
		t.Errorf("invalid prev")
	}
	//
	if tg.next.next != nil {
		t.Errorf("invalid next.next")
	}
	// 1 -> 10 -> 5 -> 2
	i = 10
	tg = addTag(head, i)
	if tg.idx != i {
		t.Errorf("expected %d", i)
	}
	//
	if head.idx != 1 || head.next.idx != 10 || head.next.next.idx != 5 || head.next.next.next.idx != 2 || head.next.next.next.next != nil {
		t.Errorf("corupted list")
	}
	//
	if invalidNextPrev(head) {
		t.Errorf("corupted list")
	}
	// 10 -> 5 -> 2
	if removeTag(head) != removeTagSlide {
		t.Error("remove failed slide")
	}
	if head.idx != 10 || head.next.idx != 5 || head.next.next.idx != 2 || head.next.next.next != nil {
		t.Errorf("corupted list")
	}
	if invalidNextPrev(head) {
		t.Errorf("corupted list")
	}
	// 10 -> 2
	if removeTag(head.next) != removeTagNormal {
		t.Error("remove failed")
	}
	if head.idx != 10 || head.next.idx != 2 || head.next.next != nil {
		t.Errorf("corupted list")
	}
	if invalidNextPrev(head) {
		t.Errorf("corupted list")
	}
	// 10
	if removeTag(head.next) != removeTagNormal {
		t.Error("remove failed")
	}
	if head.idx != 10 || head.next != nil {
		t.Errorf("corupted list")
	}
	// ->|
	if removeTag(head) != removeTagLast {
		t.Errorf("remove failed")
	}
	if head.next != nil || head.prev != nil {
		t.Errorf("remove failed")
	}
}
