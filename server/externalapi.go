package server

import (
	"bytes"
	"encoding/json"
	"github.com/skynetservices/skydns/registry"
	"log"
	"net/http"
)

func (s *Server) getRegionsHTTPHandler(w http.ResponseWriter, req *http.Request) {
	var secret string

	//read the authorization header to get the secret.
	secret = req.Header.Get("Authorization")

	if err := s.authenticate(secret); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	srv, err := s.registry.Get("any")
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

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(regions)
	w.Write(b.Bytes())

}

func (s *Server) getEnvironmentsHTTPHandler(w http.ResponseWriter, req *http.Request) {
	var secret string

	//read the authorization header to get the secret.
	secret = req.Header.Get("Authorization")

	if err := s.authenticate(secret); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	srv, err := s.registry.Get("any")
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

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(environments)
	w.Write(b.Bytes())
}

func (s *Server) getServicesHTTPHandler(w http.ResponseWriter, req *http.Request) {
	var secret string

	//read the authorization header to get the secret.
	secret = req.Header.Get("Authorization")

	if err := s.authenticate(secret); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	log.Println(req.URL.Path)
	log.Println(s.raftServer.Leader())

	var q string

	if q = req.URL.Query().Get("query"); q == "" {
		q = "any"
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

	var b bytes.Buffer
	json.NewEncoder(&b).Encode(srv)
	w.Write(b.Bytes())

}
