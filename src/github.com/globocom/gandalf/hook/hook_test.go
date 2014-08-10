// Copyright 2014 gandalf authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package hook

import (
	"github.com/tsuru/commandmocker"
	"github.com/tsuru/config"
	"github.com/tsuru/gandalf/db"
	"github.com/tsuru/gandalf/fs"
	testingfs "github.com/tsuru/tsuru/fs/testing"
	"io/ioutil"
	"launchpad.net/gocheck"
	"os"
	"strings"
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

func (s *S) TestCanCreateHookFile(c *gocheck.C) {
	hook_content := strings.NewReader("some content")
	err := createHookFile("/tmp/repositories/some-repo.git/hooks/test-can-create-hook-file", hook_content)
	c.Assert(err, gocheck.IsNil)
	file, err := fs.Filesystem().OpenFile("/tmp/repositories/some-repo.git/hooks/test-can-create-hook-file", os.O_RDONLY, 0755)
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(content), gocheck.Equals, "some content")
}

func (s *S) TestCanAddNewHook(c *gocheck.C) {
	hook_content := strings.NewReader("some content")
	err := Add("test-can-add-new-hook", []string{}, hook_content)
	c.Assert(err, gocheck.IsNil)
	file, err := fs.Filesystem().OpenFile("/home/git/bare-template/hooks/test-can-add-new-hook", os.O_RDONLY, 0755)
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(content), gocheck.Equals, "some content")
}

func (s *S) TestCanAddNewRepository(c *gocheck.C) {
	hook_content := strings.NewReader("some content")
	err := Add("test-can-add-new-repository-hook", []string{"some-repo"}, hook_content)
	c.Assert(err, gocheck.IsNil)
	file, err := fs.Filesystem().OpenFile("/tmp/repositories/some-repo.git/hooks/test-can-add-new-repository-hook", os.O_RDONLY, 0755)
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(content), gocheck.Equals, "some content")
}
