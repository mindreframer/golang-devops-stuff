// Copyright 2014 gandalf authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/tsuru/gandalf/db"
	"github.com/tsuru/gandalf/hook"
	"github.com/tsuru/gandalf/repository"
	"github.com/tsuru/gandalf/user"
	"io"
	"io/ioutil"
	"mime"
	"net/http"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
)

func accessParameters(body io.ReadCloser) (repositories, users []string, err error) {
	var params map[string][]string
	if err := parseBody(body, &params); err != nil {
		return []string{}, []string{}, err
	}
	users, ok := params["users"]
	if !ok {
		return []string{}, []string{}, errors.New("It is need a user list")
	}
	repositories, ok = params["repositories"]
	if !ok {
		return []string{}, []string{}, errors.New("It is need a repository list")
	}
	return repositories, users, nil
}

func GrantAccess(w http.ResponseWriter, r *http.Request) {
	// TODO: update README
	repositories, users, err := accessParameters(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := repository.GrantAccess(repositories, users); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "Successfully granted access to users \"%s\" into repository \"%s\"", users, repositories)
}

func RevokeAccess(w http.ResponseWriter, r *http.Request) {
	repositories, users, err := accessParameters(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := repository.RevokeAccess(repositories, users); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintf(w, "Successfully revoked access to users \"%s\" into repositories \"%s\"", users, repositories)
}

func AddKey(w http.ResponseWriter, r *http.Request) {
	keys := map[string]string{}
	if err := parseBody(r.Body, &keys); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(keys) == 0 {
		http.Error(w, "A key is needed", http.StatusBadRequest)
		return
	}
	uName := r.URL.Query().Get(":name")
	if err := user.AddKey(uName, keys); err != nil {
		switch err {
		case user.ErrInvalidKey:
			http.Error(w, err.Error(), http.StatusBadRequest)
		case user.ErrDuplicateKey:
			http.Error(w, "Key already exists.", http.StatusConflict)
		case user.ErrUserNotFound:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}
	fmt.Fprint(w, "Key(s) successfully created")
}

func RemoveKey(w http.ResponseWriter, r *http.Request) {
	uName := r.URL.Query().Get(":name")
	kName := r.URL.Query().Get(":keyname")
	if err := user.RemoveKey(uName, kName); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	fmt.Fprintf(w, "Key \"%s\" successfully removed", kName)
}

func ListKeys(w http.ResponseWriter, r *http.Request) {
	uName := r.URL.Query().Get(":name")
	keys, err := user.ListKeys(uName)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	out, err := json.Marshal(&keys)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(out)
}

type jsonUser struct {
	Name string
	Keys map[string]string
}

func NewUser(w http.ResponseWriter, r *http.Request) {
	var usr jsonUser
	if err := parseBody(r.Body, &usr); err != nil {
		http.Error(w, "Got error while parsing body: "+err.Error(), http.StatusBadRequest)
		return
	}
	u, err := user.New(usr.Name, usr.Keys)
	if err != nil {
		http.Error(w, "Got error while creating user: "+err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "User \"%s\" successfully created\n", u.Name)
}

func RemoveUser(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get(":name")
	if err := user.Remove(name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "User \"%s\" successfully removed\n", name)
}

func NewRepository(w http.ResponseWriter, r *http.Request) {
	var repo repository.Repository
	if err := parseBody(r.Body, &repo); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	rep, err := repository.New(repo.Name, repo.Users, repo.IsPublic)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "Repository \"%s\" successfully created\n", rep.Name)
}

func GetRepository(w http.ResponseWriter, r *http.Request) {
	repo, err := repository.Get(r.URL.Query().Get(":name"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	out, err := json.Marshal(&repo)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Write(out)
}

func RemoveRepository(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get(":name")
	if err := repository.Remove(name); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	fmt.Fprintf(w, "Repository \"%s\" successfully removed\n", name)
}

func RenameRepository(w http.ResponseWriter, r *http.Request) {
	var p struct{ Name string }
	defer r.Body.Close()
	err := parseBody(r.Body, &p)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}
	name := r.URL.Query().Get(":name")
	err = repository.Rename(name, p.Name)
	if err != nil && err.Error() == "not found" {
		http.Error(w, err.Error(), http.StatusNotFound)
	} else if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

type repositoryHook struct {
	Repositories []string
	Content      string
}

func AddHook(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get(":name")
	if name != "post-receive" && name != "pre-receive" && name != "update" {
		http.Error(w,
			"Unsupported hook, valid options are: post-receive, pre-receive or update",
			http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	var params repositoryHook
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	repos := []string{}
	if err := json.Unmarshal(body, &params); err != nil {
		content := strings.NewReader(string(body))
		if err := hook.Add(name, repos, content); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	content := strings.NewReader(params.Content)
	repos = params.Repositories
	if err := hook.Add(name, repos, content); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(repos) > 0 {
		fmt.Fprint(w, "hook ", name, " successfully created for ", repos, "\n")
	} else {
		fmt.Fprint(w, "hook ", name, " successfully created\n")
	}
}

func parseBody(body io.ReadCloser, result interface{}) error {
	if reflect.ValueOf(result).Kind() == reflect.Struct {
		return errors.New("parseBody function cannot deal with struct. Use pointer")
	}
	b, err := ioutil.ReadAll(body)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(b, &result); err != nil {
		return errors.New(fmt.Sprintf("Could not parse json: %s", err.Error()))
	}
	return nil
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	conn, err := db.Conn()
	if err != nil {
		return
	}
	defer conn.Close()
	if err := conn.User().Database.Session.Ping(); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "Failed to ping the database: %s\n", err)
		return
	}
	w.Write([]byte("WORKING"))
}

func GetFileContents(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get(":name")
	path := r.URL.Query().Get("path")
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		ref = "master"
	}
	if path == "" || repo == "" {
		err := fmt.Errorf("Error when trying to obtain file %s on ref %s of repository %s (repository and path are required).", path, ref, repo)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	contents, err := repository.GetFileContents(repo, ref, path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	extension := filepath.Ext(path)
	mimeType := mime.TypeByExtension(extension)
	if mimeType == "" {
		mimeType = "text/plain; charset=utf-8"
	}
	w.Header().Set("Content-Type", mimeType)
	w.Header().Set("Content-Length", strconv.Itoa(len(contents)))
	w.Write(contents)
}

func GetArchive(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get(":name")
	ref := r.URL.Query().Get("ref")
	format := r.URL.Query().Get("format")
	if ref == "" || format == "" || repo == "" {
		err := fmt.Errorf("Error when trying to obtain archive for ref '%s' (format: %s) of repository '%s' (repository, ref and format are required).", ref, format, repo)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	var archiveFormat repository.ArchiveFormat
	switch {
	case format == "tar":
		archiveFormat = repository.Tar
	case format == "tar.gz":
		archiveFormat = repository.TarGz
	default:
		archiveFormat = repository.Zip
	}
	contents, err := repository.GetArchive(repo, ref, archiveFormat)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	// Default headers
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s_%s.%s\"", repo, ref, format))
	w.Header().Set("Content-Transfer-Encoding", "binary")
	w.Header().Set("Accept-Ranges", "bytes")
	w.Header().Set("Content-Length", strconv.Itoa(len(contents)))
	// Prevent Caching of File
	w.Header().Set("Cache-Control", "private")
	w.Header().Set("Pragma", "private")
	w.Header().Set("Expires", "Mon, 26 Jul 1997 05:00:00 GMT")
	w.Write(contents)
}

func GetTree(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get(":name")
	path := r.URL.Query().Get("path")
	ref := r.URL.Query().Get("ref")
	if ref == "" {
		ref = "master"
	}
	if path == "" {
		path = "."
	}
	if repo == "" {
		err := fmt.Errorf("Error when trying to obtain tree for path %s on ref %s of repository %s (repository is required).", path, ref, repo)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tree, err := repository.GetTree(repo, ref, path)
	if err != nil {
		err := fmt.Errorf("Error when trying to obtain tree for path %s on ref %s of repository %s (%s).", path, ref, repo, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b, err := json.Marshal(tree)
	if err != nil {
		err := fmt.Errorf("Error when trying to obtain tree for path %s on ref %s of repository %s (%s).", path, ref, repo, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Write(b)
}

func GetBranches(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get(":name")
	if repo == "" {
		err := fmt.Errorf("Error when trying to obtain the branches of repository %s (repository is required).", repo)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	branches, err := repository.GetBranches(repo)
	if err != nil {
		err := fmt.Errorf("Error when trying to obtain the branches of repository %s (%s).", repo, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b, err := json.Marshal(branches)
	if err != nil {
		err := fmt.Errorf("Error when trying to obtain the branches of repository %s (%s).", repo, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Write(b)
}

func GetTags(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get(":name")
	ref := r.URL.Query().Get("ref")
	if repo == "" {
		err := fmt.Errorf("Error when trying to obtain tags on ref %s of repository %s (repository is required).", ref, repo)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tags, err := repository.GetTags(repo)
	if err != nil {
		err := fmt.Errorf("Error when trying to obtain tags on ref %s of repository %s (%s).", ref, repo, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	b, err := json.Marshal(tags)
	if err != nil {
		err := fmt.Errorf("Error when trying to obtain tags on ref %s of repository %s (%s).", ref, repo, err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.Write(b)
}

func GetDiff(w http.ResponseWriter, r *http.Request) {
	repo := r.URL.Query().Get(":name")
	previousCommit := r.URL.Query().Get("previous_commit")
	lastCommit := r.URL.Query().Get("last_commit")
	if repo == "" {
		err := fmt.Errorf("Error when trying to obtain diff between hash commits of repository %s (repository is required).", repo)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if previousCommit == "" || lastCommit == "" {
		err := fmt.Errorf("Error when trying to obtain diff between hash commits of repository %s (Hash Commit(s) are required).", repo)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	diff, err := repository.GetDiff(repo, previousCommit, lastCommit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(diff)))
	w.Write(diff)
}
