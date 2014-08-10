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

// tagItem is a holder for tags and pubsub
type tagItem struct {
	head   *tag
	pubsub pubsub
}

type tagMap struct {
	tags map[string]*tagItem
}

func (this *tagMap) init() {
	this.tags = make(map[string]*tagItem)
}

func (this *tagMap) getTag(key string) *tag {
	tagitem := this.tags[key]
	if tagitem != nil {
		return tagitem.head
	}
	return nil
}

// getAddTagItem returns tagItem by key.
// Create new tagItem and adds it to map if does not exist.
func (this *tagMap) getAddTagItem(key string) *tagItem {
	item := this.tags[key]
	if item == nil {
		item = new(tagItem)
		this.tags[key] = item
	}
	return item
}

// addTag adds tag and returns added tag and pubsub
func (this *tagMap) addTag(key string, idx int) (*tag, *pubsub) {
	item := this.getAddTagItem(key)
	if item.head == nil {
		item.head = addTag(nil, idx)
		return item.head, &item.pubsub
	}
	return addTag(item.head, idx), &item.pubsub
}

// addSubscription adds subscription and returns it
func (this *tagMap) addSubscription(key string, sub *subscription) {
	item := this.getAddTagItem(key)
	item.pubsub.add(sub)
}

// containsTag returns true only if there is a valid head for a given tagItem
func (this *tagMap) containsTag(key string) bool {
	item := this.tags[key]
	if item != nil && item.head != nil {
		return true
	}
	return false
}

// removeTag removes tagItem only if there are no active subscriptions
func (this *tagMap) removeTag(key string) {
	item := this.tags[key]
	if item != nil && item.pubsub.count() == 0 {
		delete(this.tags, key)
	}
}
