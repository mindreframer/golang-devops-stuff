// Copyright (c) 2013 Erik St. Martin, Brian Ketelsen. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package main

import (
	"sync"

	"github.com/coreos/go-etcd/etcd"
	"github.com/miekg/dns"
)

var (
	inflight     = new(single)
	etcdInflight = new(etcdSingle)
)

// Adapted from singleinflight.go from the original Go Code. Copyright 2013 The Go Authors.
type call struct {
	wg   sync.WaitGroup
	val  *dns.RRSIG
	err  error
	dups int
}

type single struct {
	sync.Mutex
	m map[string]*call
}

func (g *single) Do(key string, fn func() (*dns.RRSIG, error)) (*dns.RRSIG, error, bool) {
	g.Lock()
	if g.m == nil {
		g.m = make(map[string]*call)
	}
	if c, ok := g.m[key]; ok {
		c.dups++
		g.Unlock()
		c.wg.Wait()
		return c.val, c.err, true
	}
	c := new(call)
	c.wg.Add(1)
	g.m[key] = c
	g.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.Lock()
	delete(g.m, key)
	g.Unlock()

	return c.val, c.err, c.dups > 0
}

type etcdCall struct {
	wg   sync.WaitGroup
	val  *etcd.Response
	err  error
	dups int
}

type etcdSingle struct {
	sync.Mutex
	m map[string]*etcdCall
}

func (g *etcdSingle) Do(key string, fn func() (*etcd.Response, error)) (*etcd.Response, error, bool) {
	g.Lock()
	if g.m == nil {
		g.m = make(map[string]*etcdCall)
	}
	if c, ok := g.m[key]; ok {
		c.dups++
		g.Unlock()
		c.wg.Wait()
		return c.val, c.err, true
	}
	c := new(etcdCall)
	c.wg.Add(1)
	g.m[key] = c
	g.Unlock()

	c.val, c.err = fn()
	c.wg.Done()

	g.Lock()
	delete(g.m, key)
	g.Unlock()

	return c.val, c.err, c.dups > 0
}
