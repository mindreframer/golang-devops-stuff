/*
Declares gocheck's test suites
*/
package vulcan

import (
	"github.com/mailgun/vulcan/timeutils"
	. "launchpad.net/gocheck"
	"testing"
	"time"
)

func Test(t *testing.T) { TestingT(t) }

//This is a simple suite to use if tests dont' need anything
//special
type MainSuite struct {
	timeProvider *timeutils.FreezedTime
}

func (s *MainSuite) SetUpTest(c *C) {
	start := time.Date(2012, 3, 4, 5, 6, 7, 0, time.UTC)
	s.timeProvider = &timeutils.FreezedTime{CurrentTime: start}
}

var _ = Suite(&MainSuite{})
