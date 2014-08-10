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

package server

// pubsub
type pubsub struct {
	head *subscription
}

func (this *pubsub) hasSubscriptions() bool {
	return this.head != nil
}

func (this *pubsub) add(sub *subscription) {
	sub.next = this.head
	this.head = sub
}

type pubsubVisitor func(sub *subscription) bool

func (this *pubsub) visit(visitor pubsubVisitor) {
	prev := this.head
	for sub := this.head; sub != nil; sub = sub.next {
		if !sub.active() || !visitor(sub) {
			if sub == this.head {
				this.head = sub.next
				prev = this.head
			} else {
				prev.next = sub.next
			}
		} else {
			prev = sub
		}
	}
}

func (this *pubsub) count() int {
	i := 0
	visitor := func(sub *subscription) bool {
		i++
		return true
	}
	this.visit(visitor)
	return i
}

func (this *pubsub) publishTest(res response) {
	visitor := func(sub *subscription) bool {
		return true
	}
	this.visit(visitor)
}

// subscription represents individual client subscription
type subscription struct {
	next   *subscription // next node
	sender *responseSender
	id     uint64
}

// factory
func newSubscription(sender *responseSender, id uint64) *subscription {
	return &subscription{
		next:   nil,
		sender: sender,
		id:     id,
	}
}

//
func (this *subscription) active() bool {
	return this.sender != nil
}

//
func (this *subscription) deactivate() {
	this.sender = nil
}

//

type mapSubscriptionById map[uint64]*subscription
type mapSubscriptionByConnection map[uint64]mapSubscriptionById

func newMapSubscriptions() mapSubscriptionByConnection {
	return make(mapSubscriptionByConnection)
}

func (this *mapSubscriptionByConnection) getOrAdd(connectionId uint64) mapSubscriptionById {
	mapsub := (*this)[connectionId]
	if mapsub == nil {
		mapsub = make(mapSubscriptionById)
		(*this)[connectionId] = mapsub
	}
	return mapsub
}

func (this *mapSubscriptionByConnection) add(connectionId uint64, sub *subscription) {
	mapsub := this.getOrAdd(connectionId)
	mapsub[sub.id] = sub
}

func (this *mapSubscriptionByConnection) deactivate(connectionId uint64, pubsubid uint64) bool {
	mapsub := this.getOrAdd(connectionId)
	sub := mapsub[pubsubid]
	if sub == nil {
		return false
	}
	sub.deactivate()
	delete(mapsub, pubsubid)
	return true
}

func (this *mapSubscriptionByConnection) deactivateAll(connectionId uint64) int {
	mapsub := this.getOrAdd(connectionId)
	count := 0
	for _, sub := range mapsub {
		sub.deactivate()
		count++
	}
	delete(*this, connectionId)
	return count
}
