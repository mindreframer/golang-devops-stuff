// Copyright 2013 gandalf authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package repository

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/globocom/config"
	"github.com/globocom/gandalf/db"
	"github.com/globocom/gandalf/fs"
	"labix.org/v2/mgo"
	"labix.org/v2/mgo/bson"
	"regexp"
)

// Repository represents a Git repository. A Git repository is a record in the
// database and a directory in the filesystem (the bare repository).
type Repository struct {
	Name     string `bson:"_id"`
	Users    []string
	IsPublic bool
}

// MarshalJSON marshals the Repository in json format.
func (r *Repository) MarshalJSON() ([]byte, error) {
	data := map[string]interface{}{
		"name":    r.Name,
		"public":  r.IsPublic,
		"ssh_url": r.ReadWriteURL(),
		"git_url": r.ReadOnlyURL(),
	}
	return json.Marshal(&data)
}

// New creates a representation of a git repository. It creates a Git
// repository using the "bare-dir" setting and saves repository's meta data in
// the database.
func New(name string, users []string, isPublic bool) (*Repository, error) {
	r := &Repository{Name: name, Users: users, IsPublic: isPublic}
	if v, err := r.isValid(); !v {
		return r, err
	}
	if err := newBare(name); err != nil {
		return r, err
	}
	err := db.Session.Repository().Insert(&r)
	if mgo.IsDup(err) {
		return r, fmt.Errorf("A repository with this name already exists.")
	}
	return r, err
}

// Get find a repository by name.
func Get(name string) (Repository, error) {
	var r Repository
	err := db.Session.Repository().FindId(name).One(&r)
	return r, err
}

// Remove deletes the repository from the database and removes it's bare Git
// repository.
func Remove(name string) error {
	if err := removeBare(name); err != nil {
		return err
	}
	if err := db.Session.Repository().RemoveId(name); err != nil {
		return fmt.Errorf("Could not remove repository: %s", err)
	}
	return nil
}

// Rename renames a repository.
func Rename(oldName, newName string) error {
	repo, err := Get(oldName)
	if err != nil {
		return err
	}
	newRepo := repo
	newRepo.Name = newName
	err = db.Session.Repository().Insert(newRepo)
	if err != nil {
		return err
	}
	err = db.Session.Repository().RemoveId(oldName)
	if err != nil {
		return err
	}
	return fs.Fsystem.Rename(barePath(oldName), barePath(newName))
}

// ReadWriteURL formats the git ssh url and return it. If no remote is configured in
// gandalf.conf, this method panics.
func (r *Repository) ReadWriteURL() string {
	uid, err := config.GetString("uid")
	if err != nil {
		panic(err.Error())
	}
	remote := uid + "@%s:%s.git"
	if useSSH, _ := config.GetBool("git:ssh:use"); useSSH {
		port, err := config.GetString("git:ssh:port")
		if err == nil {
			remote = "ssh://" + uid + "@%s:" + port + "/%s.git"
		} else {
			remote = "ssh://" + uid + "@%s/%s.git"
		}
	}
	host, err := config.GetString("host")
	if err != nil {
		panic(err.Error())
	}
	return fmt.Sprintf(remote, host, r.Name)
}

// ReadOnly formats the git url and return it. If no host is configured in
// gandalf.conf, this method panics.
func (r *Repository) ReadOnlyURL() string {
	remote := "git://%s/%s.git"
	if useSSH, _ := config.GetBool("git:ssh:use"); useSSH {
		uid, err := config.GetString("uid")
		if err != nil {
			panic(err.Error())
		}
		port, err := config.GetString("git:ssh:port")
		if err == nil {
			remote = "ssh://" + uid + "@%s:" + port + "/%s.git"
		} else {
			remote = "ssh://" + uid + "@%s/%s.git"
		}
	}
	host, err := config.GetString("host")
	if err != nil {
		panic(err.Error())
	}
	return fmt.Sprintf(remote, host, r.Name)
}

// Validates a repository
// A valid repository must have:
//  - a name without any special chars only alphanumeric and underlines are allowed.
//  - at least one user in users array
func (r *Repository) isValid() (bool, error) {
	m, e := regexp.Match(`^[\w-]+$`, []byte(r.Name))
	if e != nil {
		panic(e)
	}
	if !m {
		return false, errors.New("Validation Error: repository name is not valid")
	}
	if len(r.Users) == 0 {
		return false, errors.New("Validation Error: repository should have at least one user")
	}
	return true, nil
}

// GrantAccess gives write permission for users in all specified repositories.
// If any of the repositories/users do not exists, GrantAccess just skips it.
func GrantAccess(rNames, uNames []string) error {
	_, err := db.Session.Repository().UpdateAll(bson.M{"_id": bson.M{"$in": rNames}}, bson.M{"$addToSet": bson.M{"users": bson.M{"$each": uNames}}})
	return err
}

// RevokeAccess revokes write permission from users in all specified
// repositories.
func RevokeAccess(rNames, uNames []string) error {
	_, err := db.Session.Repository().UpdateAll(bson.M{"_id": bson.M{"$in": rNames}}, bson.M{"$pullAll": bson.M{"users": uNames}})
	return err
}
