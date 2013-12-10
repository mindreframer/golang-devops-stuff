package roundrobin

import (
	"container/heap"
	. "launchpad.net/gocheck"
	"testing"
)

func TestPq(t *testing.T) { TestingT(t) }

type PQSuite struct{}

var _ = Suite(&PQSuite{})

func (s *PQSuite) TestPeek(c *C) {
	pq := &priorityQueue{}
	heap.Init(pq)

	item := &pqItem{
		c:        &cursor{hash: 1},
		priority: 5,
	}
	heap.Push(pq, item)
	c.Assert(pq.Peek().c.hash, Equals, uint32(1))
	c.Assert(pq.Len(), Equals, 1)

	item = &pqItem{
		c:        &cursor{hash: 2},
		priority: 1,
	}
	heap.Push(pq, item)
	c.Assert(pq.Len(), Equals, 2)
	c.Assert(pq.Peek().c.hash, Equals, uint32(2))
	c.Assert(pq.Peek().c.hash, Equals, uint32(2))
	c.Assert(pq.Len(), Equals, 2)

	pitem := heap.Pop(pq)
	item, ok := pitem.(*pqItem)
	if !ok {
		panic("Impossible")
	}
	c.Assert(item.c.hash, Equals, uint32(2))
	c.Assert(pq.Len(), Equals, 1)
	c.Assert(pq.Peek().c.hash, Equals, uint32(1))

	heap.Pop(pq)
	c.Assert(pq.Len(), Equals, 0)
}

func (s *PQSuite) TestUpdate(c *C) {
	pq := &priorityQueue{}
	heap.Init(pq)
	x := &pqItem{
		c:        &cursor{hash: 1},
		priority: 4,
	}
	y := &pqItem{
		c:        &cursor{hash: 2},
		priority: 3,
	}
	z := &pqItem{
		c:        &cursor{hash: 3},
		priority: 8,
	}
	heap.Push(pq, x)
	heap.Push(pq, y)
	heap.Push(pq, z)
	c.Assert(pq.Peek().c.hash, Equals, uint32(2))

	pq.Update(z, 1)
	c.Assert(pq.Peek().c.hash, Equals, uint32(3))

	pq.Update(x, 0)
	c.Assert(pq.Peek().c.hash, Equals, uint32(1))
}
