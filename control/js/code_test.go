package js

import (
	"fmt"
	"io/ioutil"
	. "launchpad.net/gocheck"
	"os"
)

type CodeSuite struct{}

var _ = Suite(&CodeSuite{})

func (s *CodeSuite) TestStringGetter(c *C) {
	code := NewStringGetter("Hello")
	out, err := code.GetCode()
	c.Assert(err, Equals, nil)
	c.Assert(out, Equals, "Hello")
}

func (s *CodeSuite) TestFileGetterSuccess(c *C) {
	filePath := fmt.Sprintf("%s/_vulcan_test_js", os.TempDir())
	err := ioutil.WriteFile(filePath, []byte("Hi"), 0666)
	c.Assert(err, Equals, nil)
	getter := NewFileGetter(filePath)
	out, err := getter.GetCode()
	c.Assert(err, Equals, nil)
	c.Assert(out, Equals, "Hi")
}

func (s *CodeSuite) TestFileGetterFailure(c *C) {
	filePath := fmt.Sprintf("%s/_vulcan_test_not_exists", os.TempDir())
	getter := NewFileGetter(filePath)
	_, err := getter.GetCode()
	c.Assert(err, Not(Equals), nil)
}
