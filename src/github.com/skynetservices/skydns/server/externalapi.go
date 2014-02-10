// Copyright (c) 2013 The SkyDNS Authors. All rights reserved.
// Use of this source code is governed by The MIT License (MIT) that can be
// found in the LICENSE file.

package server

import (
	"encoding/json"
	"github.com/skynetservices/skydns/registry"
	"log"
	"net/http"
)

func (s *Server) getRegionsHTTPHandler(w http.ResponseWriter, req *http.Request) {
	srv, err := s.registry.Get("*")
	if err != nil {
		switch err {
		case registry.ErrNotExists:
			w.Write([]byte("{}"))
		default:
			log.Println("Error: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	regions := make(map[string]int, 1)

	for _, service := range srv {
		if _, ok := regions[service.Region]; ok {
			// exists, increment
			regions[service.Region] = regions[service.Region] + 1
		} else {
			regions[service.Region] = 1
		}
	}

	if err := json.NewEncoder(w).Encode(regions); err != nil {
		log.Println("Error: ", err)
	}
}

func (s *Server) getEnvironmentsHTTPHandler(w http.ResponseWriter, req *http.Request) {
	srv, err := s.registry.Get("*")
	if err != nil {
		switch err {
		case registry.ErrNotExists:
			w.Write([]byte("{}"))
		default:
			log.Println("Error: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
		return
	}

	environments := make(map[string]int, 1)

	for _, service := range srv {
		if _, ok := environments[service.Environment]; ok {
			// exists, increment
			environments[service.Environment] = environments[service.Environment] + 1
		} else {
			environments[service.Environment] = 1
		}
	}

	if err := json.NewEncoder(w).Encode(environments); err != nil {
		log.Println("Error: ", err)
	}
}

func (s *Server) getServicesHTTPHandler(w http.ResponseWriter, req *http.Request) {
	log.Println(req.URL.Path)
	log.Println(s.raftServer.Leader())

	var q string

	if q = req.URL.Query().Get("query"); q == "" {
		q = "*"
	}

	log.Println("Retrieving All Services for query", q)

	srv, err := s.registry.Get(q)

	if err != nil {
		switch err {
		case registry.ErrNotExists:
			http.Error(w, err.Error(), http.StatusNotFound)
		default:
			log.Println("Error: ", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	if err := json.NewEncoder(w).Encode(srv); err != nil {
		log.Println("Error: ", err)
	}
}
