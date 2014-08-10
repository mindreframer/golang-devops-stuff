// Copyright (c) 2014 The SkyDNS Authors. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package main

import (
	"log"
	"net/url"
	"strings"

	"github.com/coreos/go-etcd/etcd"
)

func NewClient(machines []string) (client *etcd.Client) {
	// set default if not specified in env
	if len(machines) == 1 && machines[0] == "" {
		machines[0] = "http://127.0.0.1:4001"
	}
	if strings.HasPrefix(machines[0], "https://") {
		var err error
		// TODO(miek): machines is local, the rest is global, ugly.
		if client, err = etcd.NewTLSClient(machines, tlspem, tlskey, cacert); err != nil {
			// TODO(miek): would be nice if this wasn't a fatal error
			log.Fatalf("failure to connect: %s\n", err)
		}
		client.SyncCluster()
	} else {
		client = etcd.NewClient(machines)
		client.SyncCluster()
	}
	return client
}

// updateClient updates the client with the machines found in v2/_etcd/machines.
func (s *server) UpdateClient(resp *etcd.Response) {
	machines := make([]string, 0, 3)
	for _, m := range resp.Node.Nodes {
		u, e := url.Parse(m.Value)
		if e != nil {
			continue
		}
		// etcd=bla&raft=bliep
		// TODO(miek): surely there is a better way to do this
		ms := strings.Split(u.String(), "&")
		if len(ms) == 0 {
			continue
		}
		if len(ms[0]) < 5 {
			continue
		}
		machines = append(machines, ms[0][5:])
	}
	s.config.log.Infof("setting new etcd cluster to %v", machines)
	c := NewClient(machines)
	// This is our RCU, switch the pointer, old readers get the old
	// one, new reader get the new one.
	s.client = c
}

// get is a wrapper for client.Get that uses SingleInflight to suppress multiple
// outstanding queries.
func get(client *etcd.Client, path string, recursive bool) (*etcd.Response, error) {
	resp, err, _ := etcdInflight.Do(path, func() (*etcd.Response, error) {
		r, e := client.Get(path, false, recursive)
		if e != nil {
			return nil, e
		}
		return r, e
	})
	if err != nil {
		return resp, err
	}
	// shared?
	return resp, err
}
