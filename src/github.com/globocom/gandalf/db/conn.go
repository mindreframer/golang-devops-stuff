// Copyright 2013 gandalf authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// Package db provides util functions to deal with Gandalf's database.
package db

import (
	"github.com/globocom/config"
	"labix.org/v2/mgo"
)

type session struct {
	DB *mgo.Database
}

// The global Session that must be used by users.
var Session session

// Connect uses database:url and database:name settings in config file and
// connects to the database. If it cannot connect or these settings are not
// defined, it will panic.
func Connect() {
	url, _ := config.GetString("database:url")
	if url == "" {
		url = "localhost:27017"
	}
	name, _ := config.GetString("database:name")
	if name == "" {
		name = "gandalf"
	}
	s, err := mgo.Dial(url)
	if err != nil {
		panic(err)
	}
	Session.DB = s.DB(name)
	Session.DB.Session.LiveServers()
}

// Repository returns a reference to the "repository" collection in MongoDB.
func (s *session) Repository() *mgo.Collection {
	return s.DB.C("repository")
}

// User returns a reference to the "user" collection in MongoDB.
func (s *session) User() *mgo.Collection {
	return s.DB.C("user")
}

func (s *session) Key() *mgo.Collection {
	index := mgo.Index{Key: []string{"body"}, Unique: true}
	c := s.DB.C("key")
	c.EnsureIndex(index)
	return c
}
