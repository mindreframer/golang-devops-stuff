package server

import (
	"encoding/json"
	"github.com/goraft/raft"
	"github.com/gorilla/mux"
	"github.com/skynetservices/skydns/msg"
	"github.com/skynetservices/skydns/registry"
	"log"
	"net/http"
	"strings"
)

// Handle API add callback requests.
func (s *Server) addCallbackHTTPHandler(w http.ResponseWriter, req *http.Request) {
	vars := mux.Vars(req)

	var uuid string
	var ok bool
	var secret string

	secret = req.Header.Get("Authorization")
	if err := s.authenticate(secret); err != nil {
		http.Error(w, err.Error(), http.StatusForbidden)
		return
	}

	if uuid, ok = vars["uuid"]; !ok {
		http.Error(w, "UUID required", http.StatusBadRequest)
		return
	}

	var cb msg.Callback

	if err := json.NewDecoder(req.Body).Decode(&cb); err != nil {
		log.Println("Error: ", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	cb.UUID = uuid
	// Lookup the service(s)
	// TODO: this should be a function call (or method)
	key := cb.Region + "." + strings.Replace(cb.Version, ".", "-", -1) + "." + cb.Name + "." + cb.Environment
	key = strings.ToLower(key)
	services, err := s.registry.Get(key)
	if err != nil || len(services) == 0 {
		log.Println("Service not found for callback", key)
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	// Reset to save memory, only used so find the services(s).
	cb.Name = ""
	cb.Version = ""
	cb.Environment = ""
	cb.Region = ""
	cb.Host = ""

	notExists := 0
	for _, serv := range services {
		if _, err := s.raftServer.Do(NewAddCallbackCommand(serv, cb)); err != nil {
			switch err {
			case registry.ErrNotExists:
				notExists++
				continue
			case raft.NotLeaderError:
				s.redirectToLeader(w, req)
				return
			default:
				log.Println("Error: ", err)
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
	}
	if notExists == len(services) {
		http.Error(w, registry.ErrNotExists.Error(), http.StatusNotFound)
		return
	}
	w.WriteHeader(http.StatusCreated)
}
