package main

import (
	"html/template"
	"net/http"
)


const (
	STARTING_STATUS      = "starting"
	STARTED_STATUS       = "started"
	STOPPING_STATUS      = "stopping"
	STOPPED_STATUS       = "stopped"
	ERROR_STATUS         = "error"
	NA_STATUS            = "n/a"
	PASSIVATED_STATUS    = "passivated"
)

type Status struct {
	alive    string
	current  string
	expected string
	service *Service
}

func (s *Status) equals(other *Status) bool {
	if(s == nil && other == nil) {
		return true
	}
	return s!= nil && other != nil && s.alive == other.alive &&
	s.current == other.current &&
	s.expected == other.expected
}

func (s *Status) compute() string {

	if s != nil {
		alive := s.alive
		expected := s.expected
		current := s.current
		switch current {
		case STOPPED_STATUS:
			if expected == PASSIVATED_STATUS {
				return PASSIVATED_STATUS
			}else if expected == STOPPED_STATUS {
				return STOPPED_STATUS
			} else {
				return ERROR_STATUS
			}
		case STARTING_STATUS:
			if expected == STARTED_STATUS {
				return STARTING_STATUS
			} else {
				return ERROR_STATUS
			}
		case STARTED_STATUS:
			if alive != "" {
				if expected != STARTED_STATUS {
					return ERROR_STATUS
				}
				return STARTED_STATUS
			} else {
				return ERROR_STATUS
			}
		case STOPPING_STATUS:
			if expected == STOPPED_STATUS {
				return STOPPED_STATUS
			} else {
				return ERROR_STATUS
			}
			// N/A
		default:
			return ERROR_STATUS
		}
	}

	return STARTED_STATUS
}


type StatusError struct {
	computedStatus string
	status         *Status
}

func (s StatusError) Error() string {
	return s.computedStatus
}

type StatusPage struct {
	config *Config
	error  StatusError
}

type StatusData struct {
	status string
}

func (sp *StatusPage) serve(w http.ResponseWriter, r *http.Request) {

	var code int
	switch sp.error.computedStatus {
	case "notfound":
		code = http.StatusNotFound
	case STARTING_STATUS, PASSIVATED_STATUS:
		code = http.StatusServiceUnavailable
	default:
		code = http.StatusInternalServerError

	}

	templateDir := sp.config.templateDir
	tmpl, err := template.ParseFiles(templateDir+"/main.tpl", templateDir+"/body_"+sp.error.computedStatus+".tpl")
	if err != nil {
		http.Error(w, "Unable to serve page : " + sp.error.computedStatus, code)

		return
	}


	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(code)
	err = tmpl.ExecuteTemplate(w, "main", &StatusData{sp.error.computedStatus})
	if err != nil {
		http.Error(w, "Failed to execute templates : "+err.Error(), 500)
	}
}
