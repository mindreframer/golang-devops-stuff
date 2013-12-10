package roundrobin

import (
	. "launchpad.net/gocheck"
	"testing"
)

func TestCursor(t *testing.T) { TestingT(t) }

type CursorSuite struct{}

var _ = Suite(&CursorSuite{})

func (s *CursorSuite) TestHashing(c *C) {
	a, b, z := &E{id: "a", active: true}, &E{id: "b", active: true}, &E{id: "c", active: true}

	cr := newCursor(endpoints(a, b, z))
	cr2 := newCursor(endpoints(a, b, z))
	cr3 := newCursor(endpoints(a, b))

	c.Assert(cr.hash, Equals, cr2.hash)
	c.Assert(cr.hash, Not(Equals), cr3.hash)
}

func (s *CursorSuite) TestCursorBasics(c *C) {
	a, b, z := &E{id: "a", active: true}, &E{id: "b", active: true}, &E{id: "c", active: true}

	cm := newCursorMap()
	cr := cm.addCursor(endpoints(a, b, z))
	cr2 := cm.addCursor(endpoints(a, b))

	c.Assert(cm.getCursor(endpoints(a, b, z)), Equals, cr)
	c.Assert(cm.getCursor(endpoints(a, b)), Equals, cr2)

	var nilCursor *cursor
	c.Assert(cm.getCursor(endpoints(a)), Equals, nilCursor)

	// Make sure it deletes just what we need, no more
	cm.deleteCursor(cr)
	c.Assert(cm.getCursor(endpoints(a, b, z)), Equals, nilCursor)
	c.Assert(cm.getCursor(endpoints(a, b)), Equals, cr2)

	// Deleting same cursor works fine
	err := cm.deleteCursor(cr)
	c.Assert(err, NotNil)

	cm.deleteCursor(cr2)
	c.Assert(cm.getCursor(endpoints(a, b)), Equals, nilCursor)
}

func (s *CursorSuite) TestCursorCollision(c *C) {
	a, b, z := &E{id: "a", active: true}, &E{id: "b", active: true}, &E{id: "c", active: true}

	cm := newCursorMap()
	cr := cm.addCursor(endpoints(a, b, z))
	cr2 := cm.addCursor(endpoints(a, b, z))

	// make sure there is a collision
	c.Assert(len(cm.cursors[cr.hash]), Equals, 2)

	// despite of collision, we are still able to fetch real cursor
	c.Assert(cm.getCursor(endpoints(a, b, z)), Equals, cr)

	// delete cursor and make sure we did not delete the other one
	cm.deleteCursor(cr)
	c.Assert(len(cm.cursors[cr.hash]), Equals, 1)
	cm.deleteCursor(cr2)
	c.Assert(len(cm.cursors[cr.hash]), Equals, 0)

	// deleting nonexistent cursor wont hurt us
	cm.deleteCursor(cr2)
	c.Assert(len(cm.cursors[cr.hash]), Equals, 0)
}
