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

import (
	"fmt"
	"testing"
)

func validateQueueItem(t *testing.T, item *queueItem, expected int) {
	got := 0
	for ; item != nil; got++ {
		item = item.next
	}
	ASSERT_TRUE(t, expected == got, fmt.Sprintf("expected %v got %v", expected, got))
}

func TestQueue(t *testing.T) {
	que := newQueue("name")
	que.push("a")
	que.push("b")
	que.push("c")
	que.push("d")
	que.push("e")
	que.push("f")

	item, removed := que.pop(1)
	ASSERT_TRUE(t, removed == 1, "unexpected removed")
	validateQueueItem(t, item, removed)
	ASSERT_TRUE(t, item.data == "a", "unexpected queueItem data")
	ASSERT_TRUE(t, item.next == nil, "unexpected queueItem next")
	ASSERT_TRUE(t, que.front != nil, "unexpected queue front")
	ASSERT_TRUE(t, que.back != nil, "unexpected queue back")

	item, removed = que.pop(2)
	ASSERT_TRUE(t, removed == 2, "unexpected removed")
	validateQueueItem(t, item, removed)
	ASSERT_TRUE(t, item.data == "b", "unexpected queueItem data")
	ASSERT_TRUE(t, item.next.data == "c", "unexpected queueItem data")
	ASSERT_TRUE(t, item.next.next == nil, "unexpected queueItem next")
	ASSERT_TRUE(t, que.front != nil, "unexpected queue front")
	ASSERT_TRUE(t, que.back != nil, "unexpected queue back")

	item, removed = que.pop(3)
	ASSERT_TRUE(t, removed == 3, "unexpected removed")
	validateQueueItem(t, item, removed)
	ASSERT_TRUE(t, item.data == "d", "unexpected queueItem data")
	ASSERT_TRUE(t, item.next.data == "e", "unexpected queueItem data")
	ASSERT_TRUE(t, item.next.next.data == "f", "unexpected queueItem data")
	ASSERT_TRUE(t, item.next.next.next == nil, "unexpected queueItem next")
	ASSERT_TRUE(t, que.front == nil, "unexpected queue front")
	ASSERT_TRUE(t, que.back == nil, "unexpected queue back")

	//

	que.push("a")
	que.push("b")

	item, removed = que.pop(1)
	ASSERT_TRUE(t, removed == 1, "unexpected removed")
	validateQueueItem(t, item, removed)
	ASSERT_TRUE(t, item.data == "a", "unexpected queueItem data")
	ASSERT_TRUE(t, item.next == nil, "unexpected queueItem next")
	ASSERT_TRUE(t, que.front != nil, "unexpected queue front")
	ASSERT_TRUE(t, que.back != nil, "unexpected queue back")

	item, removed = que.pop(1)
	ASSERT_TRUE(t, removed == 1, "unexpected removed")
	validateQueueItem(t, item, removed)
	ASSERT_TRUE(t, item.data == "b", "unexpected queueItem data")
	ASSERT_TRUE(t, item.next == nil, "unexpected queueItem next")
	ASSERT_TRUE(t, que.front == nil, "unexpected queue front")
	ASSERT_TRUE(t, que.back == nil, "unexpected queue back")
}
