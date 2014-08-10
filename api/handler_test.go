// Copyright 2014 gandalf authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/tsuru/gandalf/db"
	"github.com/tsuru/gandalf/fs"
	"github.com/tsuru/gandalf/repository"
	"github.com/tsuru/gandalf/user"
	"gopkg.in/mgo.v2/bson"
	"io"
	"io/ioutil"
	"launchpad.net/gocheck"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"strings"
)

type bufferCloser struct {
	*bytes.Buffer
}

func (b bufferCloser) Close() error { return nil }

func get(url string, b io.Reader, c *gocheck.C) (*httptest.ResponseRecorder, *http.Request) {
	return request("GET", url, b, c)
}

func post(url string, b io.Reader, c *gocheck.C) (*httptest.ResponseRecorder, *http.Request) {
	return request("POST", url, b, c)
}

func del(url string, b io.Reader, c *gocheck.C) (*httptest.ResponseRecorder, *http.Request) {
	return request("DELETE", url, b, c)
}

func request(method, url string, b io.Reader, c *gocheck.C) (*httptest.ResponseRecorder, *http.Request) {
	request, err := http.NewRequest(method, url, b)
	c.Assert(err, gocheck.IsNil)
	request.Header.Set("Content-Type", "application/json")
	recorder := httptest.NewRecorder()
	return recorder, request
}

func readBody(b io.Reader, c *gocheck.C) string {
	body, err := ioutil.ReadAll(b)
	c.Assert(err, gocheck.IsNil)
	return string(body)
}

func (s *S) authKeysContent(c *gocheck.C) string {
	authKeysPath := path.Join(os.Getenv("HOME"), ".ssh", "authorized_keys")
	f, err := fs.Filesystem().OpenFile(authKeysPath, os.O_RDWR|os.O_EXCL, 0755)
	c.Assert(err, gocheck.IsNil)
	content, err := ioutil.ReadAll(f)
	return string(content)
}

func (s *S) TestNewUser(c *gocheck.C) {
	b := strings.NewReader(fmt.Sprintf(`{"name": "brain", "keys": {"keyname": %q}}`, rawKey))
	recorder, request := post("/user", b, c)
	NewUser(recorder, request)
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	defer conn.User().Remove(bson.M{"_id": "brain"})
	defer conn.Key().Remove(bson.M{"username": "brain"})
	body, err := ioutil.ReadAll(recorder.Body)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(body), gocheck.Equals, "User \"brain\" successfully created\n")
	c.Assert(recorder.Code, gocheck.Equals, 200)
}

func (s *S) TestNewUserShouldSaveInDB(c *gocheck.C) {
	b := strings.NewReader(`{"name": "brain", "keys": {"content": "some id_rsa.pub key.. use your imagination!", "name": "somekey"}}`)
	recorder, request := post("/user", b, c)
	NewUser(recorder, request)
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	defer conn.User().Remove(bson.M{"_id": "brain"})
	defer conn.Key().Remove(bson.M{"username": "brain"})
	var u user.User
	err = conn.User().Find(bson.M{"_id": "brain"}).One(&u)
	c.Assert(err, gocheck.IsNil)
	c.Assert(u.Name, gocheck.Equals, "brain")
}

func (s *S) TestNewUserShouldRepassParseBodyErrors(c *gocheck.C) {
	b := strings.NewReader("{]9afe}")
	recorder, request := post("/user", b, c)
	NewUser(recorder, request)
	body := readBody(recorder.Body, c)
	expected := "Got error while parsing body: Could not parse json: invalid character ']' looking for beginning of object key string"
	got := strings.Replace(body, "\n", "", -1)
	c.Assert(got, gocheck.Equals, expected)
}

func (s *S) TestNewUserShouldRequireUserName(c *gocheck.C) {
	b := strings.NewReader(`{"name": ""}`)
	recorder, request := post("/user", b, c)
	NewUser(recorder, request)
	body := readBody(recorder.Body, c)
	expected := "Got error while creating user: Validation Error: user name is not valid"
	got := strings.Replace(body, "\n", "", -1)
	c.Assert(got, gocheck.Equals, expected)
}

func (s *S) TestNewUserWihoutKeys(c *gocheck.C) {
	b := strings.NewReader(`{"name": "brain"}`)
	recorder, request := post("/user", b, c)
	NewUser(recorder, request)
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	defer conn.User().Remove(bson.M{"_id": "brain"})
	c.Assert(recorder.Code, gocheck.Equals, 200)
}

func (s *S) TestGetRepository(c *gocheck.C) {
	r := repository.Repository{Name: "onerepo"}
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	err = conn.Repository().Insert(&r)
	c.Assert(err, gocheck.IsNil)
	defer conn.Repository().Remove(bson.M{"_id": r.Name})
	recorder, request := get("/repository/onerepo?:name=onerepo", nil, c)
	GetRepository(recorder, request)
	body, err := ioutil.ReadAll(recorder.Body)
	c.Assert(err, gocheck.IsNil)
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	c.Assert(err, gocheck.IsNil)
	expected := map[string]interface{}{
		"name":    r.Name,
		"public":  r.IsPublic,
		"ssh_url": r.ReadWriteURL(),
		"git_url": r.ReadOnlyURL(),
	}
	c.Assert(data, gocheck.DeepEquals, expected)
}

func (s *S) TestGetRepositoryDoesNotExist(c *gocheck.C) {
	recorder, request := get("/repository/doesnotexists?:name=doesnotexists", nil, c)
	GetRepository(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, 500)
}

func (s *S) TestNewRepository(c *gocheck.C) {
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	defer conn.Repository().Remove(bson.M{"_id": "some_repository"})
	b := strings.NewReader(`{"name": "some_repository", "users": ["r2d2"]}`)
	recorder, request := post("/repository", b, c)
	NewRepository(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "Repository \"some_repository\" successfully created\n"
	c.Assert(got, gocheck.Equals, expected)
}

func (s *S) TestNewRepositoryShouldSaveInDB(c *gocheck.C) {
	b := strings.NewReader(`{"name": "myRepository", "users": ["r2d2"]}`)
	recorder, request := post("/repository", b, c)
	NewRepository(recorder, request)
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	collection := conn.Repository()
	defer collection.Remove(bson.M{"_id": "myRepository"})
	var p repository.Repository
	err = collection.Find(bson.M{"_id": "myRepository"}).One(&p)
	c.Assert(err, gocheck.IsNil)
}

func (s *S) TestNewRepositoryShouldSaveUserIdInRepository(c *gocheck.C) {
	b := strings.NewReader(`{"name": "myRepository", "users": ["r2d2", "brain"]}`)
	recorder, request := post("/repository", b, c)
	NewRepository(recorder, request)
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	collection := conn.Repository()
	defer collection.Remove(bson.M{"_id": "myRepository"})
	var p repository.Repository
	err = collection.Find(bson.M{"_id": "myRepository"}).One(&p)
	c.Assert(err, gocheck.IsNil)
	c.Assert(len(p.Users), gocheck.Not(gocheck.Equals), 0)
}

func (s *S) TestNewRepositoryShouldReturnErrorWhenNoUserIsPassed(c *gocheck.C) {
	b := strings.NewReader(`{"name": "myRepository"}`)
	recorder, request := post("/repository", b, c)
	NewRepository(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, 400)
	body := readBody(recorder.Body, c)
	expected := "Validation Error: repository should have at least one user"
	got := strings.Replace(body, "\n", "", -1)
	c.Assert(got, gocheck.Equals, expected)
}

func (s *S) TestNewRepositoryShouldReturnErrorWhenNoParametersArePassed(c *gocheck.C) {
	b := strings.NewReader("{}")
	recorder, request := post("/repository", b, c)
	NewRepository(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, 400)
	body := readBody(recorder.Body, c)
	expected := "Validation Error: repository name is not valid"
	got := strings.Replace(body, "\n", "", -1)
	c.Assert(got, gocheck.Equals, expected)
}

func (s *S) TestParseBodyShouldMapBodyJsonToGivenStruct(c *gocheck.C) {
	var p repository.Repository
	b := bufferCloser{bytes.NewBufferString(`{"name": "Dummy Repository"}`)}
	err := parseBody(b, &p)
	c.Assert(err, gocheck.IsNil)
	expected := "Dummy Repository"
	c.Assert(p.Name, gocheck.Equals, expected)
}

func (s *S) TestParseBodyShouldReturnErrorWhenJsonIsInvalid(c *gocheck.C) {
	var p repository.Repository
	b := bufferCloser{bytes.NewBufferString("{]ja9aW}")}
	err := parseBody(b, &p)
	c.Assert(err, gocheck.NotNil)
	expected := "Could not parse json: invalid character ']' looking for beginning of object key string"
	c.Assert(err.Error(), gocheck.Equals, expected)
}

func (s *S) TestParseBodyShouldReturnErrorWhenBodyIsEmpty(c *gocheck.C) {
	var p repository.Repository
	b := bufferCloser{bytes.NewBufferString("")}
	err := parseBody(b, &p)
	c.Assert(err, gocheck.NotNil)
	c.Assert(err, gocheck.ErrorMatches, `^Could not parse json:.*$`)
}

func (s *S) TestParseBodyShouldReturnErrorWhenResultParamIsNotAPointer(c *gocheck.C) {
	var p repository.Repository
	b := bufferCloser{bytes.NewBufferString(`{"name": "something"}`)}
	err := parseBody(b, p)
	c.Assert(err, gocheck.NotNil)
	expected := "parseBody function cannot deal with struct. Use pointer"
	c.Assert(err.Error(), gocheck.Equals, expected)
}

func (s *S) TestNewRepositoryShouldReturnErrorWhenBodyIsEmpty(c *gocheck.C) {
	b := strings.NewReader("")
	recorder, request := post("/repository", b, c)
	NewRepository(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, 400)
}

func (s *S) TestGrantAccessUpdatesReposDocument(c *gocheck.C) {
	u, err := user.New("pippin", map[string]string{})
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	defer conn.User().Remove(bson.M{"_id": "pippin"})
	c.Assert(err, gocheck.IsNil)
	r := repository.Repository{Name: "onerepo"}
	err = conn.Repository().Insert(&r)
	c.Assert(err, gocheck.IsNil)
	defer conn.Repository().Remove(bson.M{"_id": r.Name})
	r2 := repository.Repository{Name: "otherepo"}
	err = conn.Repository().Insert(&r2)
	c.Assert(err, gocheck.IsNil)
	defer conn.Repository().Remove(bson.M{"_id": r2.Name})
	b := bytes.NewBufferString(fmt.Sprintf(`{"repositories": ["%s", "%s"], "users": ["%s"]}`, r.Name, r2.Name, u.Name))
	rec, req := del("/repository/grant", b, c)
	GrantAccess(rec, req)
	var repos []repository.Repository
	err = conn.Repository().Find(bson.M{"_id": bson.M{"$in": []string{r.Name, r2.Name}}}).All(&repos)
	c.Assert(err, gocheck.IsNil)
	c.Assert(rec.Code, gocheck.Equals, 200)
	for _, repo := range repos {
		c.Assert(repo.Users, gocheck.DeepEquals, []string{u.Name})
	}
}

func (s *S) TestRevokeAccessUpdatesReposDocument(c *gocheck.C) {
	r := repository.Repository{Name: "onerepo", Users: []string{"Umi", "Luke"}}
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	err = conn.Repository().Insert(&r)
	c.Assert(err, gocheck.IsNil)
	defer conn.Repository().Remove(bson.M{"_id": r.Name})
	r2 := repository.Repository{Name: "otherepo", Users: []string{"Umi", "Luke"}}
	err = conn.Repository().Insert(&r2)
	c.Assert(err, gocheck.IsNil)
	defer conn.Repository().Remove(bson.M{"_id": r2.Name})
	b := bytes.NewBufferString(fmt.Sprintf(`{"repositories": ["%s", "%s"], "users": ["Umi"]}`, r.Name, r2.Name))
	rec, req := del("/repository/revoke", b, c)
	RevokeAccess(rec, req)
	var repos []repository.Repository
	err = conn.Repository().Find(bson.M{"_id": bson.M{"$in": []string{r.Name, r2.Name}}}).All(&repos)
	c.Assert(err, gocheck.IsNil)
	for _, repo := range repos {
		c.Assert(repo.Users, gocheck.DeepEquals, []string{"Luke"})
	}
}

func (s *S) TestAddKey(c *gocheck.C) {
	usr, err := user.New("Frodo", map[string]string{})
	c.Assert(err, gocheck.IsNil)
	defer user.Remove(usr.Name)
	b := strings.NewReader(fmt.Sprintf(`{"keyname": %q}`, rawKey))
	recorder, request := post(fmt.Sprintf("/user/%s/key?:name=%s", usr.Name, usr.Name), b, c)
	AddKey(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "Key(s) successfully created"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, 200)
	var k user.Key
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	err = conn.Key().Find(bson.M{"name": "keyname", "username": usr.Name}).One(&k)
	c.Assert(err, gocheck.IsNil)
	c.Assert(k.Body, gocheck.Equals, keyBody)
	c.Assert(k.Comment, gocheck.Equals, keyComment)
}

func (s *S) TestAddPostReceiveHookRepository(c *gocheck.C) {
	b := strings.NewReader(`{"repositories": ["some-repo"], "content": "some content"}`)
	recorder, request := post("repository/hook/post-receive?:name=post-receive", b, c)
	AddHook(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "hook post-receive successfully created for [some-repo]\n"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, 200)
	file, err := fs.Filesystem().OpenFile("/tmp/repositories/some-repo.git/hooks/post-receive", os.O_RDONLY, 0755)
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(content), gocheck.Equals, "some content")
}

func (s *S) TestAddPreReceiveHookRepository(c *gocheck.C) {
	b := strings.NewReader(`{"repositories": ["some-repo"], "content": "some content"}`)
	recorder, request := post("repository/hook/pre-receive?:name=pre-receive", b, c)
	AddHook(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "hook pre-receive successfully created for [some-repo]\n"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, 200)
	file, err := fs.Filesystem().OpenFile("/tmp/repositories/some-repo.git/hooks/pre-receive", os.O_RDONLY, 0755)
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(content), gocheck.Equals, "some content")
}

func (s *S) TestAddUpdateReceiveHookRepository(c *gocheck.C) {
	b := strings.NewReader(`{"repositories": ["some-repo"], "content": "some content"}`)
	recorder, request := post("repository/hook/update?:name=update", b, c)
	AddHook(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "hook update successfully created for [some-repo]\n"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, 200)
	file, err := fs.Filesystem().OpenFile("/tmp/repositories/some-repo.git/hooks/update", os.O_RDONLY, 0755)
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(content), gocheck.Equals, "some content")
}

func (s *S) TestAddInvalidHookRepository(c *gocheck.C) {
	b := strings.NewReader(`{"repositories": ["some-repo"], "content": "some content"}`)
	recorder, request := post("repository/hook/invalid-hook?:name=invalid-hook", b, c)
	AddHook(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "Unsupported hook, valid options are: post-receive, pre-receive or update\n"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, 400)
}

func (s *S) TestAddPostReceiveHook(c *gocheck.C) {
	b := strings.NewReader(`{"content": "some content"}`)
	recorder, request := post("/hook/post-receive?:name=post-receive", b, c)
	AddHook(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "hook post-receive successfully created\n"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, 200)
	file, err := fs.Filesystem().OpenFile("/home/git/bare-template/hooks/post-receive", os.O_RDONLY, 0755)
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(content), gocheck.Equals, "some content")
}

func (s *S) TestAddPreReceiveHook(c *gocheck.C) {
	b := strings.NewReader(`{"content": "some content"}`)
	recorder, request := post("/hook/pre-receive?:name=pre-receive", b, c)
	AddHook(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "hook pre-receive successfully created\n"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, 200)
	file, err := fs.Filesystem().OpenFile("/home/git/bare-template/hooks/pre-receive", os.O_RDONLY, 0755)
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(content), gocheck.Equals, "some content")
}

func (s *S) TestAddUpdateHook(c *gocheck.C) {
	b := strings.NewReader(`{"content": "some content"}`)
	recorder, request := post("/hook/update?:name=update", b, c)
	AddHook(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "hook update successfully created\n"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, 200)
	file, err := fs.Filesystem().OpenFile("/home/git/bare-template/hooks/update", os.O_RDONLY, 0755)
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(content), gocheck.Equals, "some content")
}

func (s *S) TestAddInvalidHook(c *gocheck.C) {
	b := strings.NewReader(`{"content": "some content"}`)
	recorder, request := post("/hook/invalid-hook?:name=invalid-hook", b, c)
	AddHook(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "Unsupported hook, valid options are: post-receive, pre-receive or update\n"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, 400)
}

func (s *S) TestAddPostReceiveOldFormatHook(c *gocheck.C) {
	b := strings.NewReader("some content")
	recorder, request := post("/hook/post-receive?:name=post-receive", b, c)
	AddHook(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "hook post-receive successfully created\n"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, 200)
	file, err := fs.Filesystem().OpenFile("/home/git/bare-template/hooks/post-receive", os.O_RDONLY, 0755)
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(content), gocheck.Equals, "some content")
}

func (s *S) TestAddPreReceiveOldFormatHook(c *gocheck.C) {
	b := strings.NewReader("some content")
	recorder, request := post("/hook/pre-receive?:name=pre-receive", b, c)
	AddHook(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "hook pre-receive successfully created\n"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, 200)
	file, err := fs.Filesystem().OpenFile("/home/git/bare-template/hooks/pre-receive", os.O_RDONLY, 0755)
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(content), gocheck.Equals, "some content")
}

func (s *S) TestAddUpdateOldFormatHook(c *gocheck.C) {
	b := strings.NewReader("some content")
	recorder, request := post("/hook/update?:name=update", b, c)
	AddHook(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "hook update successfully created\n"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, 200)
	file, err := fs.Filesystem().OpenFile("/home/git/bare-template/hooks/update", os.O_RDONLY, 0755)
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(content), gocheck.Equals, "some content")
}

func (s *S) TestAddInvalidOldFormatHook(c *gocheck.C) {
	b := strings.NewReader("some content")
	recorder, request := post("/hook/invalid-hook?:name=invalid-hook", b, c)
	AddHook(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "Unsupported hook, valid options are: post-receive, pre-receive or update\n"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, 400)
}

func (s *S) TestAddKeyShouldReturnErrorWhenUserDoesNotExists(c *gocheck.C) {
	b := strings.NewReader(`{"key": "a public key"}`)
	recorder, request := post("/user/Frodo/key?:name=Frodo", b, c)
	AddKey(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, 404)
	body, err := ioutil.ReadAll(recorder.Body)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(body), gocheck.Equals, "User not found\n")
}

func (s *S) TestAddKeyShouldReturnProperStatusCodeWhenKeyAlreadyExists(c *gocheck.C) {
	usr, err := user.New("Frodo", map[string]string{"keyname": rawKey})
	c.Assert(err, gocheck.IsNil)
	defer user.Remove(usr.Name)
	b := strings.NewReader(fmt.Sprintf(`{"keyname": %q}`, rawKey))
	recorder, request := post(fmt.Sprintf("/user/%s/key?:name=%s", usr.Name, usr.Name), b, c)
	AddKey(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "Key already exists.\n"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusConflict)
}

func (s *S) TestAddKeyShouldNotAcceptRepeatedKeysForDifferentUsers(c *gocheck.C) {
	usr, err := user.New("Frodo", map[string]string{"keyname": rawKey})
	c.Assert(err, gocheck.IsNil)
	defer user.Remove(usr.Name)
	usr2, err := user.New("tempo", nil)
	c.Assert(err, gocheck.IsNil)
	defer user.Remove(usr2.Name)
	b := strings.NewReader(fmt.Sprintf(`{"keyname": %q}`, rawKey))
	recorder, request := post(fmt.Sprintf("/user/%s/key?:name=%s", usr2.Name, usr2.Name), b, c)
	AddKey(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "Key already exists.\n"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusConflict)
}

func (s *S) TestAddKeyInvalidKey(c *gocheck.C) {
	u := user.User{Name: "Frodo"}
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	err = conn.User().Insert(&u)
	c.Assert(err, gocheck.IsNil)
	defer conn.User().Remove(bson.M{"_id": "Frodo"})
	b := strings.NewReader(`{"keyname":"invalid-rsa"}`)
	recorder, request := post(fmt.Sprintf("/user/%s/key?:name=%s", u.Name, u.Name), b, c)
	AddKey(recorder, request)
	got := readBody(recorder.Body, c)
	expected := "Invalid key\n"
	c.Assert(got, gocheck.Equals, expected)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusBadRequest)
}

func (s *S) TestAddKeyShouldRequireKey(c *gocheck.C) {
	u := user.User{Name: "Frodo"}
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	err = conn.User().Insert(&u)
	c.Assert(err, gocheck.IsNil)
	defer conn.User().Remove(bson.M{"_id": "Frodo"})
	b := strings.NewReader(`{}`)
	recorder, request := post("/user/Frodo/key?:name=Frodo", b, c)
	AddKey(recorder, request)
	body := readBody(recorder.Body, c)
	expected := "A key is needed"
	got := strings.Replace(body, "\n", "", -1)
	c.Assert(got, gocheck.Equals, expected)
}

func (s *S) TestAddKeyShouldWriteKeyInAuthorizedKeysFile(c *gocheck.C) {
	u := user.User{Name: "Frodo"}
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	err = conn.User().Insert(&u)
	c.Assert(err, gocheck.IsNil)
	defer conn.User().RemoveId("Frodo")
	b := strings.NewReader(fmt.Sprintf(`{"key": "%s"}`, rawKey))
	recorder, request := post("/user/Frodo/key?:name=Frodo", b, c)
	AddKey(recorder, request)
	defer conn.Key().Remove(bson.M{"name": "key", "username": u.Name})
	c.Assert(recorder.Code, gocheck.Equals, 200)
	content := s.authKeysContent(c)
	c.Assert(strings.HasSuffix(strings.TrimSpace(content), rawKey), gocheck.Equals, true)
}

func (s *S) TestRemoveKeyGivesExpectedSuccessResponse(c *gocheck.C) {
	u, err := user.New("Gandalf", map[string]string{"keyname": rawKey})
	c.Assert(err, gocheck.IsNil)
	defer user.Remove(u.Name)
	url := "/user/Gandalf/key/keyname?:keyname=keyname&:name=Gandalf"
	recorder, request := del(url, nil, c)
	RemoveKey(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, 200)
	b := readBody(recorder.Body, c)
	c.Assert(b, gocheck.Equals, `Key "keyname" successfully removed`)
}

func (s *S) TestRemoveKeyRemovesKeyFromDatabase(c *gocheck.C) {
	u, err := user.New("Gandalf", map[string]string{"keyname": rawKey})
	c.Assert(err, gocheck.IsNil)
	defer user.Remove(u.Name)
	url := "/user/Gandalf/key/keyname?:keyname=keyname&:name=Gandalf"
	recorder, request := del(url, nil, c)
	RemoveKey(recorder, request)
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	count, err := conn.Key().Find(bson.M{"name": "keyname", "username": "Gandalf"}).Count()
	c.Assert(err, gocheck.IsNil)
	c.Assert(count, gocheck.Equals, 0)
}

func (s *S) TestRemoveKeyShouldRemoveKeyFromAuthorizedKeysFile(c *gocheck.C) {
	u, err := user.New("Gandalf", map[string]string{"keyname": rawKey})
	c.Assert(err, gocheck.IsNil)
	defer user.Remove(u.Name)
	url := "/user/Gandalf/key/keyname?:keyname=keyname&:name=Gandalf"
	recorder, request := del(url, nil, c)
	RemoveKey(recorder, request)
	content := s.authKeysContent(c)
	c.Assert(content, gocheck.Equals, "")
}

func (s *S) TestRemoveKeyShouldReturnErrorWithLineBreakAtEnd(c *gocheck.C) {
	url := "/user/idiocracy/key/keyname?:keyname=keyname&:name=idiocracy"
	recorder, request := del(url, nil, c)
	RemoveKey(recorder, request)
	b := readBody(recorder.Body, c)
	c.Assert(b, gocheck.Equals, "User not found\n")
}

func (s *S) TestListKeysGivesExpectedSuccessResponse(c *gocheck.C) {
	keys := map[string]string{"key1": rawKey, "key2": otherKey}
	u, err := user.New("Gandalf", keys)
	c.Assert(err, gocheck.IsNil)
	defer user.Remove(u.Name)
	url := "/user/Gandalf/keys?:name=Gandalf"
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	ListKeys(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, 200)
	body, err := ioutil.ReadAll(recorder.Body)
	c.Assert(err, gocheck.IsNil)
	var data map[string]string
	err = json.Unmarshal(body, &data)
	c.Assert(err, gocheck.IsNil)
	c.Assert(data, gocheck.DeepEquals, keys)
}

func (s *S) TestListKeysWithoutKeysGivesEmptyJSON(c *gocheck.C) {
	u, err := user.New("Gandalf", map[string]string{})
	c.Assert(err, gocheck.IsNil)
	defer user.Remove(u.Name)
	url := "/user/Gandalf/keys?:name=Gandalf"
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	ListKeys(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, 200)
	b := readBody(recorder.Body, c)
	c.Assert(b, gocheck.Equals, "{}")
}

func (s *S) TestListKeysWithInvalidUserReturnsNotFound(c *gocheck.C) {
	url := "/user/no-Gandalf/keys?:name=no-Gandalf"
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	ListKeys(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, 404)
	b := readBody(recorder.Body, c)
	c.Assert(b, gocheck.Equals, "User not found\n")
}

func (s *S) TestRemoveUser(c *gocheck.C) {
	u, err := user.New("username", map[string]string{})
	c.Assert(err, gocheck.IsNil)
	url := fmt.Sprintf("/user/%s/?:name=%s", u.Name, u.Name)
	request, err := http.NewRequest("DELETE", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	RemoveUser(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, 200)
	b, err := ioutil.ReadAll(recorder.Body)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(b), gocheck.Equals, "User \"username\" successfully removed\n")
}

func (s *S) TestRemoveUserShouldRemoveFromDB(c *gocheck.C) {
	u, err := user.New("anuser", map[string]string{})
	c.Assert(err, gocheck.IsNil)
	url := fmt.Sprintf("/user/%s/?:name=%s", u.Name, u.Name)
	request, err := http.NewRequest("DELETE", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	RemoveUser(recorder, request)
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	collection := conn.User()
	lenght, err := collection.Find(bson.M{"_id": u.Name}).Count()
	c.Assert(err, gocheck.IsNil)
	c.Assert(lenght, gocheck.Equals, 0)
}

func (s *S) TestRemoveRepository(c *gocheck.C) {
	r, err := repository.New("myRepo", []string{"pippin"}, true)
	c.Assert(err, gocheck.IsNil)
	url := fmt.Sprintf("repository/%s/?:name=%s", r.Name, r.Name)
	request, err := http.NewRequest("DELETE", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	RemoveRepository(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, 200)
	b, err := ioutil.ReadAll(recorder.Body)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(b), gocheck.Equals, "Repository \"myRepo\" successfully removed\n")
}

func (s *S) TestRemoveRepositoryShouldRemoveFromDB(c *gocheck.C) {
	r, err := repository.New("myRepo", []string{"pippin"}, true)
	c.Assert(err, gocheck.IsNil)
	url := fmt.Sprintf("repository/%s/?:name=%s", r.Name, r.Name)
	request, err := http.NewRequest("DELETE", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	RemoveRepository(recorder, request)
	conn, err := db.Conn()
	c.Assert(err, gocheck.IsNil)
	defer conn.Close()
	err = conn.Repository().Find(bson.M{"_id": r.Name}).One(&r)
	c.Assert(err, gocheck.ErrorMatches, "^not found$")
}

func (s *S) TestRemoveRepositoryShouldReturn400OnFailure(c *gocheck.C) {
	url := fmt.Sprintf("repository/%s/?:name=%s", "foo", "foo")
	request, err := http.NewRequest("DELETE", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	RemoveRepository(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, 400)
}

func (s *S) TestRemoveRepositoryShouldReturnErrorMsgWhenRepoDoesNotExists(c *gocheck.C) {
	url := fmt.Sprintf("repository/%s/?:name=%s", "foo", "foo")
	request, err := http.NewRequest("DELETE", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	RemoveRepository(recorder, request)
	b, err := ioutil.ReadAll(recorder.Body)
	c.Assert(err, gocheck.IsNil)
	c.Assert(string(b), gocheck.Equals, "Could not remove repository: not found\n")
}

func (s *S) TestRenameRepository(c *gocheck.C) {
	r, err := repository.New("raising", []string{"guardian@what.com"}, true)
	c.Assert(err, gocheck.IsNil)
	url := fmt.Sprintf("/repository/%s/?:name=%s", r.Name, r.Name)
	body := strings.NewReader(`{"name":"freedom"}`)
	request, err := http.NewRequest("PUT", url, body)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	RenameRepository(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusOK)
	_, err = repository.Get("raising")
	c.Assert(err, gocheck.NotNil)
	r.Name = "freedom"
	repo, err := repository.Get("freedom")
	c.Assert(err, gocheck.IsNil)
	c.Assert(repo, gocheck.DeepEquals, *r)
}

func (s *S) TestRenameRepositoryInvalidJSON(c *gocheck.C) {
	url := "/repository/foo/?:name=foo"
	body := strings.NewReader(`{"name""`)
	request, err := http.NewRequest("PUT", url, body)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	RenameRepository(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusBadRequest)
}

func (s *S) TestRenameRepositoryNotfound(c *gocheck.C) {
	url := "/repository/foo/?:name=foo"
	body := strings.NewReader(`{"name":"freedom"}`)
	request, err := http.NewRequest("PUT", url, body)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	RenameRepository(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusNotFound)
}

func (s *S) TestHealthcheck(c *gocheck.C) {
	url := "/healthcheck"
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	HealthCheck(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusOK)
	c.Assert(recorder.Body.String(), gocheck.Equals, "WORKING")
}

func (s *S) TestGetFileContents(c *gocheck.C) {
	url := "/repository/repo/contents?:name=repo&path=README.txt"
	expected := "result"
	repository.Retriever = &repository.MockContentRetriever{
		ResultContents: []byte(expected),
	}
	defer func() {
		repository.Retriever = nil
	}()
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetFileContents(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusOK)
	c.Assert(recorder.Body.String(), gocheck.Equals, expected)
	c.Assert(recorder.Header()["Content-Type"][0], gocheck.Equals, "text/plain; charset=utf-8")
	c.Assert(recorder.Header()["Content-Length"][0], gocheck.Equals, "6")
}

func (s *S) TestGetFileContentsWithoutExtension(c *gocheck.C) {
	url := "/repository/repo/contents?:name=repo&path=README"
	expected := "result"
	repository.Retriever = &repository.MockContentRetriever{
		ResultContents: []byte(expected),
	}
	defer func() {
		repository.Retriever = nil
	}()
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetFileContents(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusOK)
	c.Assert(recorder.Body.String(), gocheck.Equals, expected)
	c.Assert(recorder.Header()["Content-Type"][0], gocheck.Equals, "text/plain; charset=utf-8")
	c.Assert(recorder.Header()["Content-Length"][0], gocheck.Equals, "6")
}

func (s *S) TestGetFileContentsWithRef(c *gocheck.C) {
	url := "/repository/repo/contents?:name=repo&path=README.txt&ref=other"
	expected := "result"
	mockRetriever := repository.MockContentRetriever{
		ResultContents: []byte(expected),
	}
	repository.Retriever = &mockRetriever
	defer func() {
		repository.Retriever = nil
	}()
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetFileContents(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusOK)
	c.Assert(recorder.Body.String(), gocheck.Equals, expected)
	c.Assert(recorder.Header()["Content-Type"][0], gocheck.Equals, "text/plain; charset=utf-8")
	c.Assert(recorder.Header()["Content-Length"][0], gocheck.Equals, "6")
	c.Assert(mockRetriever.LastRef, gocheck.Equals, "other")
}

func (s *S) TestGetFileContentsWhenCommandFails(c *gocheck.C) {
	url := "/repository/repo/contents?:name=repo&path=README.txt&ref=other"
	outputError := fmt.Errorf("command error")
	repository.Retriever = &repository.MockContentRetriever{
		OutputError: outputError,
	}
	defer func() {
		repository.Retriever = nil
	}()
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetFileContents(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusNotFound)
	c.Assert(recorder.Body.String(), gocheck.Equals, "command error\n")
}

func (s *S) TestGetFileContentsWhenNoRepository(c *gocheck.C) {
	url := "/repository//contents?:name=&path=README.txt&ref=other"
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetFileContents(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusBadRequest)
	expected := "Error when trying to obtain file README.txt on ref other of repository  (repository and path are required).\n"
	c.Assert(recorder.Body.String(), gocheck.Equals, expected)
}

func (s *S) TestGetArchiveWhenNoRef(c *gocheck.C) {
	url := "/repository/repo/archive?:name=repo&ref=&format=zip"
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetArchive(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusBadRequest)
	expected := "Error when trying to obtain archive for ref '' (format: zip) of repository 'repo' (repository, ref and format are required).\n"
	c.Assert(recorder.Body.String(), gocheck.Equals, expected)
}

func (s *S) TestGetArchiveWhenNoRepo(c *gocheck.C) {
	url := "/repository//archive?:name=&ref=master&format=zip"
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetArchive(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusBadRequest)
	expected := "Error when trying to obtain archive for ref 'master' (format: zip) of repository '' (repository, ref and format are required).\n"
	c.Assert(recorder.Body.String(), gocheck.Equals, expected)
}

func (s *S) TestGetArchiveWhenNoFormat(c *gocheck.C) {
	url := "/repository/repo/archive?:name=repo&ref=master&format="
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetArchive(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusBadRequest)
	expected := "Error when trying to obtain archive for ref 'master' (format: ) of repository 'repo' (repository, ref and format are required).\n"
	c.Assert(recorder.Body.String(), gocheck.Equals, expected)
}

func (s *S) TestGetArchiveWhenCommandFails(c *gocheck.C) {
	url := "/repository/repo/archive?:name=repo&ref=master&format=zip"
	expected := fmt.Errorf("output error")
	mockRetriever := repository.MockContentRetriever{
		OutputError: expected,
	}
	repository.Retriever = &mockRetriever
	defer func() {
		repository.Retriever = nil
	}()
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetArchive(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusNotFound)
	c.Assert(recorder.Body.String(), gocheck.Equals, "output error\n")
}

func (s *S) TestGetArchive(c *gocheck.C) {
	url := "/repository/repo/archive?:name=repo&ref=master&format=zip"
	expected := "result123"
	mockRetriever := repository.MockContentRetriever{
		ResultContents: []byte(expected),
	}
	repository.Retriever = &mockRetriever
	defer func() {
		repository.Retriever = nil
	}()
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetArchive(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusOK)
	c.Assert(recorder.Body.String(), gocheck.Equals, expected)
	c.Assert(mockRetriever.LastFormat, gocheck.Equals, repository.Zip)
	c.Assert(recorder.Header()["Content-Type"][0], gocheck.Equals, "application/octet-stream")
	c.Assert(recorder.Header()["Content-Disposition"][0], gocheck.Equals, "attachment; filename=\"repo_master.zip\"")
	c.Assert(recorder.Header()["Content-Transfer-Encoding"][0], gocheck.Equals, "binary")
	c.Assert(recorder.Header()["Accept-Ranges"][0], gocheck.Equals, "bytes")
	c.Assert(recorder.Header()["Content-Length"][0], gocheck.Equals, "9")
	c.Assert(recorder.Header()["Cache-Control"][0], gocheck.Equals, "private")
	c.Assert(recorder.Header()["Pragma"][0], gocheck.Equals, "private")
	c.Assert(recorder.Header()["Expires"][0], gocheck.Equals, "Mon, 26 Jul 1997 05:00:00 GMT")
}

func (s *S) TestGetTreeWithDefaultValues(c *gocheck.C) {
	url := "/repository/repo/tree?:name=repo"
	tree := make([]map[string]string, 1)
	tree[0] = make(map[string]string)
	tree[0]["permission"] = "333"
	tree[0]["filetype"] = "blob"
	tree[0]["hash"] = "123456"
	tree[0]["path"] = "filename.txt"
	tree[0]["rawPath"] = "raw/filename.txt"
	mockRetriever := repository.MockContentRetriever{
		Tree: tree,
	}
	repository.Retriever = &mockRetriever
	defer func() {
		repository.Retriever = nil
	}()
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetTree(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusOK)
	var obj []map[string]string
	json.Unmarshal(recorder.Body.Bytes(), &obj)
	c.Assert(len(obj), gocheck.Equals, 1)
	c.Assert(obj[0]["permission"], gocheck.Equals, tree[0]["permission"])
	c.Assert(obj[0]["filetype"], gocheck.Equals, tree[0]["filetype"])
	c.Assert(obj[0]["hash"], gocheck.Equals, tree[0]["hash"])
	c.Assert(obj[0]["path"], gocheck.Equals, tree[0]["path"])
	c.Assert(obj[0]["rawPath"], gocheck.Equals, tree[0]["rawPath"])
	c.Assert(mockRetriever.LastRef, gocheck.Equals, "master")
	c.Assert(mockRetriever.LastPath, gocheck.Equals, ".")
}

func (s *S) TestGetTreeWithSpecificPath(c *gocheck.C) {
	url := "/repository/repo/tree?:name=repo&path=/test"
	tree := make([]map[string]string, 1)
	tree[0] = make(map[string]string)
	tree[0]["permission"] = "333"
	tree[0]["filetype"] = "blob"
	tree[0]["hash"] = "123456"
	tree[0]["path"] = "/test/filename.txt"
	tree[0]["rawPath"] = "/test/raw/filename.txt"
	mockRetriever := repository.MockContentRetriever{
		Tree: tree,
	}
	repository.Retriever = &mockRetriever
	defer func() {
		repository.Retriever = nil
	}()
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetTree(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusOK)
	var obj []map[string]string
	json.Unmarshal(recorder.Body.Bytes(), &obj)
	c.Assert(len(obj), gocheck.Equals, 1)
	c.Assert(obj[0]["permission"], gocheck.Equals, tree[0]["permission"])
	c.Assert(obj[0]["filetype"], gocheck.Equals, tree[0]["filetype"])
	c.Assert(obj[0]["hash"], gocheck.Equals, tree[0]["hash"])
	c.Assert(obj[0]["path"], gocheck.Equals, tree[0]["path"])
	c.Assert(obj[0]["rawPath"], gocheck.Equals, tree[0]["rawPath"])
	c.Assert(mockRetriever.LastRef, gocheck.Equals, "master")
	c.Assert(mockRetriever.LastPath, gocheck.Equals, "/test")
}

func (s *S) TestGetTreeWithSpecificRef(c *gocheck.C) {
	url := "/repository/repo/tree?:name=repo&path=/test&ref=1.1.1"
	tree := make([]map[string]string, 1)
	tree[0] = make(map[string]string)
	tree[0]["permission"] = "333"
	tree[0]["filetype"] = "blob"
	tree[0]["hash"] = "123456"
	tree[0]["path"] = "/test/filename.txt"
	tree[0]["rawPath"] = "/test/raw/filename.txt"
	mockRetriever := repository.MockContentRetriever{
		Tree: tree,
	}
	repository.Retriever = &mockRetriever
	defer func() {
		repository.Retriever = nil
	}()
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetTree(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusOK)
	var obj []map[string]string
	json.Unmarshal(recorder.Body.Bytes(), &obj)
	c.Assert(len(obj), gocheck.Equals, 1)
	c.Assert(obj[0]["permission"], gocheck.Equals, tree[0]["permission"])
	c.Assert(obj[0]["filetype"], gocheck.Equals, tree[0]["filetype"])
	c.Assert(obj[0]["hash"], gocheck.Equals, tree[0]["hash"])
	c.Assert(obj[0]["path"], gocheck.Equals, tree[0]["path"])
	c.Assert(obj[0]["rawPath"], gocheck.Equals, tree[0]["rawPath"])
	c.Assert(mockRetriever.LastRef, gocheck.Equals, "1.1.1")
	c.Assert(mockRetriever.LastPath, gocheck.Equals, "/test")
}

func (s *S) TestGetTreeWhenNoRepo(c *gocheck.C) {
	url := "/repository//tree?:name="
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetTree(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusBadRequest)
	expected := "Error when trying to obtain tree for path . on ref master of repository  (repository is required).\n"
	c.Assert(recorder.Body.String(), gocheck.Equals, expected)
}

func (s *S) TestGetTreeWhenCommandFails(c *gocheck.C) {
	url := "/repository/repo/tree/?:name=repo&ref=master&path=/test"
	expected := fmt.Errorf("output error")
	mockRetriever := repository.MockContentRetriever{
		OutputError: expected,
	}
	repository.Retriever = &mockRetriever
	defer func() {
		repository.Retriever = nil
	}()
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetTree(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusBadRequest)
	c.Assert(recorder.Body.String(), gocheck.Equals, "Error when trying to obtain tree for path /test on ref master of repository repo (output error).\n")
}

func (s *S) TestGetBranches(c *gocheck.C) {
	url := "/repository/repo/branches?:name=repo"
	refs := make([]repository.Ref, 1)
	refs[0] = repository.Ref{
		Ref:       "a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9",
		Name:      "doge_barks",
		CreatedAt: "Mon Jul 28 10:13:27 2014 -0300",
		Committer: &repository.GitUser{
			Name:  "doge",
			Email: "<much@email.com>",
		},
		Author: &repository.GitUser{
			Name:  "doge",
			Email: "<much@email.com>",
		},
		Subject: "will bark",
		Links: &repository.Links{
			ZipArchive: repository.GetArchiveUrl("repo", "doge_barks", "zip"),
			TarArchive: repository.GetArchiveUrl("repo", "doge_barks", "tar.gz"),
		},
	}
	mockRetriever := repository.MockContentRetriever{
		Refs: refs,
	}
	repository.Retriever = &mockRetriever
	defer func() {
		repository.Retriever = nil
	}()
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetBranches(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusOK)
	var obj []repository.Ref
	json.Unmarshal(recorder.Body.Bytes(), &obj)
	c.Assert(obj, gocheck.HasLen, 1)
	c.Assert(obj[0], gocheck.DeepEquals, refs[0])
}

func (s *S) TestGetBranchesWhenRepoNotSupplied(c *gocheck.C) {
	url := "/repository//branches?:name="
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetBranches(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusBadRequest)
	expected := "Error when trying to obtain the branches of repository  (repository is required).\n"
	c.Assert(recorder.Body.String(), gocheck.Equals, expected)
}

func (s *S) TestGetBranchesWhenRepoNonExistent(c *gocheck.C) {
	url := "/repository/repo/branches?:name=repo"
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetBranches(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusBadRequest)
	expected := "Error when trying to obtain the branches of repository repo (Error when trying to obtain the refs of repository repo (Repository does not exist).).\n"
	c.Assert(recorder.Body.String(), gocheck.Equals, expected)
}

func (s *S) TestGetBranchesWhenCommandFails(c *gocheck.C) {
	url := "/repository/repo/branches/?:name=repo"
	expected := fmt.Errorf("output error")
	mockRetriever := repository.MockContentRetriever{
		OutputError: expected,
	}
	repository.Retriever = &mockRetriever
	defer func() {
		repository.Retriever = nil
	}()
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetBranches(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusBadRequest)
	c.Assert(recorder.Body.String(), gocheck.Equals, "Error when trying to obtain the branches of repository repo (output error).\n")
}

func (s *S) TestGetTags(c *gocheck.C) {
	url := "/repository/repo/tags?:name=repo"
	refs := make([]repository.Ref, 1)
	refs[0] = repository.Ref{
		Ref:       "a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9",
		Name:      "doge_barks",
		CreatedAt: "Mon Jul 28 10:13:27 2014 -0300",
		Committer: &repository.GitUser{
			Name:  "doge",
			Email: "<much@email.com>",
		},
		Author: &repository.GitUser{
			Name:  "doge",
			Email: "<much@email.com>",
		},
		Subject: "will bark",
		Links: &repository.Links{
			ZipArchive: repository.GetArchiveUrl("repo", "doge_barks", "zip"),
			TarArchive: repository.GetArchiveUrl("repo", "doge_barks", "tar.gz"),
		},
	}
	mockRetriever := repository.MockContentRetriever{
		Refs: refs,
	}
	repository.Retriever = &mockRetriever
	defer func() {
		repository.Retriever = nil
	}()
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetTags(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusOK)
	var obj []repository.Ref
	json.Unmarshal(recorder.Body.Bytes(), &obj)
	c.Assert(obj, gocheck.HasLen, 1)
	c.Assert(obj[0], gocheck.DeepEquals, refs[0])
}

func (s *S) TestGetDiff(c *gocheck.C) {
	url := "/repository/repo/diff/commits?:name=repo&previous_commit=1b970b076bbb30d708e262b402d4e31910e1dc10&last_commit=545b1904af34458704e2aa06ff1aaffad5289f8f"
	expected := "test_diff"
	repository.Retriever = &repository.MockContentRetriever{
		ResultContents: []byte(expected),
	}
	defer func() {
		repository.Retriever = nil
	}()
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetDiff(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusOK)
	c.Assert(recorder.Body.String(), gocheck.Equals, expected)
}

func (s *S) TestGetDiffWhenCommandFails(c *gocheck.C) {
	url := "/repository/repo/diff/commits?:name=repo&previous_commit=1b970b076bbb30d708e262b402d4e31910e1dc10&last_commit=545b1904af34458704e2aa06ff1aaffad5289f8f"
	outputError := fmt.Errorf("command error")
	repository.Retriever = &repository.MockContentRetriever{
		OutputError: outputError,
	}
	defer func() {
		repository.Retriever = nil
	}()
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetDiff(recorder, request)
	c.Assert(recorder.Code, gocheck.Equals, http.StatusNotFound)
	c.Assert(recorder.Body.String(), gocheck.Equals, "command error\n")
}

func (s *S) TestGetDiffWhenNoRepository(c *gocheck.C) {
	url := "/repository//diff/commits?:name=&previous_commit=1b970b076bbb30d708e262b402d4e31910e1dc10&last_commit=545b1904af34458704e2aa06ff1aaffad5289f8f"
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetDiff(recorder, request)
	expected := "Error when trying to obtain diff between hash commits of repository  (repository is required).\n"
	c.Assert(recorder.Code, gocheck.Equals, http.StatusBadRequest)
	c.Assert(recorder.Body.String(), gocheck.Equals, expected)
}

func (s *S) TestGetDiffWhenNoCommits(c *gocheck.C) {
	url := "/repository/repo/diff/commits?:name=repo&previous_commit=&last_commit="
	request, err := http.NewRequest("GET", url, nil)
	c.Assert(err, gocheck.IsNil)
	recorder := httptest.NewRecorder()
	GetDiff(recorder, request)
	expected := "Error when trying to obtain diff between hash commits of repository repo (Hash Commit(s) are required).\n"
	c.Assert(recorder.Code, gocheck.Equals, http.StatusBadRequest)
	c.Assert(recorder.Body.String(), gocheck.Equals, expected)
}
