package roundrobin

import (
	"container/heap"
)

// An Item is something we manage in a priority queue.
type pqItem struct {
	c        *cursor // cursor of the round robin iterators
	priority int     // The priority of the item in the queue.
	// The index is needed by update and is maintained by the heap.Interface methods.
	index int // The index of the item in the heap.
}

// A PriorityQueue implements heap.Interface and holds Items.
type priorityQueue []*pqItem

func (pq priorityQueue) Len() int { return len(pq) }

func (pq priorityQueue) Less(i, j int) bool {
	return pq[i].priority < pq[j].priority
}

func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *priorityQueue) Push(x interface{}) {
	n := len(*pq)
	item := x.(*pqItem)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *priorityQueue) Pop() interface{} {
	old := *pq
	n := len(old)
	item := old[n-1]
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (pq *priorityQueue) Peek() *pqItem {
	items := *pq
	return items[0]
}

// update modifies the priority and value of an Item in the queue.
func (pq *priorityQueue) Update(item *pqItem, priority int) {
	heap.Remove(pq, item.index)
	item.priority = priority
	heap.Push(pq, item)
}
