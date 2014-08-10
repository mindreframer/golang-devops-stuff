/* Copyright (C) 2014 CompleteDB LLC.
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

// queueItem
type queueItem struct {
	next *queueItem
	data string
}

func newQueueItem(data string) *queueItem {
	return &queueItem{
		next: nil,
		data: data,
	}
}

// queue
type queue struct {
	name  string
	front *queueItem
	back  *queueItem
}

// table factory
func newQueue(name string) *queue {
	que := &queue{
		name:  name,
		front: nil,
	}
	return que
}

func (this *queue) push(data string) {
	item := newQueueItem(data)
	if this.front == nil {
		this.front = item
		this.back = item
		return
	}
	this.back.next = item
	this.back = this.back.next
}

// depth indicates max number of items  to be removed from the queue
// returns number of items removed
func (this *queue) pop(depth int) (*queueItem, int) {
	removed := 0
	item := this.front
	last := item
	for last != nil {
		removed++
		depth--
		if depth < 1 {
			break
		}
		last = last.next
	}
	this.front = nil
	if last != nil {
		this.front = last.next
		last.next = nil
	}
	if last == this.back {
		this.back = nil
	}
	return item, removed
}
