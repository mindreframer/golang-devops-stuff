// Copyright 2014 gandalf authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package db

import (
	"github.com/tsuru/config"
	"launchpad.net/gocheck"
	"testing"
)

func Test(t *testing.T) { gocheck.TestingT(t) }

type S struct{}

var _ = gocheck.Suite(&S{})

func (s *S) SetUpSuite(c *gocheck.C) {
	config.Set("database:url", "127.0.0.1:27017")
	config.Set("database:name", "gandalf_tests")
}

func (s *S) TearDownSuite(c *gocheck.C) {
	conn, err := Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	conn.User().Database.DropDatabase()
}

func (s *S) TestSessionRepositoryShouldReturnAMongoCollection(c *gocheck.C) {
	conn, err := Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	rep := conn.Repository()
	cRep := conn.Collection("repository")
	c.Assert(rep, gocheck.DeepEquals, cRep)
}

func (s *S) TestSessionUserShouldReturnAMongoCollection(c *gocheck.C) {
	conn, err := Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	usr := conn.User()
	cUsr := conn.Collection("user")
	c.Assert(usr, gocheck.DeepEquals, cUsr)
}

func (s *S) TestSessionKeyShouldReturnKeyCollection(c *gocheck.C) {
	conn, err := Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	key := conn.Key()
	cKey := conn.Collection("key")
	c.Assert(key, gocheck.DeepEquals, cKey)
}

func (s *S) TestSessionKeyBodyIsUnique(c *gocheck.C) {
	conn, err := Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	key := conn.Key()
	indexes, err := key.Indexes()
	c.Assert(err, gocheck.IsNil)
	c.Assert(indexes, gocheck.HasLen, 2)
	c.Assert(indexes[1].Key, gocheck.DeepEquals, []string{"body"})
	c.Assert(indexes[1].Unique, gocheck.DeepEquals, true)
}

func (s *S) TestConnect(c *gocheck.C) {
	conn, err := Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	c.Assert(conn.User().Database.Name, gocheck.Equals, "gandalf_tests")
	err = conn.User().Database.Session.Ping()
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestConnectDefaultSettings(c *gocheck.C) {
	oldURL, _ := config.Get("database:url")
	defer config.Set("database:url", oldURL)
	oldName, _ := config.Get("database:name")
	defer config.Set("database:name", oldName)
	config.Unset("database:url")
	config.Unset("database:name")
	conn, err := Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	c.Assert(conn.User().Database.Name, gocheck.Equals, "gandalf")
	c.Assert(conn.User().Database.Session.LiveServers(), gocheck.DeepEquals, []string{"127.0.0.1:27017"})
}
