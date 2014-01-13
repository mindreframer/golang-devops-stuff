package command

import (
	. "launchpad.net/gocheck"
)

type ErrorSuite struct{}

var _ = Suite(&ErrorSuite{})

func (s *ErrorSuite) TestRetry(c *C) {
	r := &RetryError{Seconds: 1}
	c.Assert(r.Error(), Not(Equals), nil)
}

func (s *ErrorSuite) Test(c *C) {
	d := &AllUpstreamsDownError{}
	c.Assert(d.Error(), Not(Equals), nil)
}
