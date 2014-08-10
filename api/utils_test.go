// Copyright 2014 gandalf authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"github.com/tsuru/commandmocker"
	"github.com/tsuru/config"
	"github.com/tsuru/gandalf/db"
	"github.com/tsuru/gandalf/fs"
	"github.com/tsuru/gandalf/user"
	testingfs "github.com/tsuru/tsuru/fs/testing"
	"gopkg.in/mgo.v2/bson"
	"launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { gocheck.TestingT(t) }

type S struct {
	tmpdir string
	rfs    *testingfs.RecordingFs
}

var _ = gocheck.Suite(&S{})

func (s *S) SetUpSuite(c *gocheck.C) {
	err := config.ReadConfigFile("../etc/gandalf.conf")
	c.Assert(err, gocheck.IsNil)
	config.Set("database:url", "127.0.0.1:27017")
	config.Set("database:name", "gandalf_api_tests")
	s.tmpdir, err = commandmocker.Add("git", "")
	c.Assert(err, gocheck.IsNil)
}

func (s *S) SetUpTest(c *gocheck.C) {
	s.rfs = &testingfs.RecordingFs{}
	fs.Fsystem = s.rfs
	bareTemplate, _ := config.GetString("git:bare:template")
	fs.Fsystem.MkdirAll(bareTemplate+"/hooks", 0755)
}

func (s *S) TearDownTest(c *gocheck.C) {
	fs.Fsystem = nil
}

func (s *S) TearDownSuite(c *gocheck.C) {
	commandmocker.Remove(s.tmpdir)
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	conn.User().Database.DropDatabase()
}

func (s *S) TestGetUserOr404(c *gocheck.C) {
	u := user.User{Name: "umi"}
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	err = conn.User().Insert(&u)
	c.Assert(err, gocheck.IsNil)
	defer conn.User().Remove(bson.M{"_id": u.Name})
	rUser, err := getUserOr404("umi")
	c.Assert(err, gocheck.IsNil)
	c.Assert(rUser.Name, gocheck.Equals, "umi")
}

func (s *S) TestGetUserOr404ShouldReturn404WhenUserDoesntExists(c *gocheck.C) {
	_, e := getUserOr404("umi")
	expected := "User umi not found"
	got := e.Error()
	c.Assert(got, gocheck.Equals, expected)
}
