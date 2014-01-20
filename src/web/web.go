package web

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"logger"
	"net/http"
	"plugins/record"
	"query"
	"strings"
	"types"
)

var httpMethodDispatch = map[string]func(*WebHandler, http.ResponseWriter, *http.Request){
	"GET":  (*WebHandler).DispatchGET,
	"POST": (*WebHandler).DispatchPOST,
	"PUT":  (*WebHandler).DispatchPUT,
}

type Request struct {
	Name  string
	Value interface{}
}

type WebHandler struct {
	Config types.CirconusConfig
	Logger *logger.Logger
}

func (wh *WebHandler) showUnauthorized(w http.ResponseWriter) {
	w.Header().Add("WWW-Authenticate", "Basic realm=\"gollector\"")
	w.WriteHeader(401)
}

func (wh *WebHandler) handleAuth(r *http.Request) bool {
	header, ok := r.Header["Authorization"]

	if !ok {
		return false
	}

	decoded, err := base64.StdEncoding.DecodeString(strings.Split(header[0], " ")[1])

	if err != nil {
		return false
	}

	credentials := strings.Split(string(decoded), ":")

	if credentials[0] != wh.Config.Username || credentials[1] != wh.Config.Password {
		return false
	}

	return true
}

func (wh *WebHandler) readAndUnmarshal(w http.ResponseWriter, r *http.Request, requestType string) Request {
	req := Request{}
	in, err := ioutil.ReadAll(r.Body)

	wh.Logger.Log("debug", fmt.Sprintf("Handling %s with payload '%s'", requestType, in))

	if err != nil {
		wh.Logger.Log("crit", fmt.Sprintf("Error encountered reading: %s", err))
		w.WriteHeader(500)
	}

	json.Unmarshal(in, &req)

	return req
}

func (wh *WebHandler) DispatchGET(w http.ResponseWriter, r *http.Request) {
	wh.Logger.Log("debug", "Handling GET")

	out, err := json.Marshal(query.GetResults())

	if err != nil {
		wh.Logger.Log("crit", fmt.Sprintf("Error marshalling all metrics: %s", err))
		w.WriteHeader(500)
	} else {
		wh.Logger.Log("debug", fmt.Sprintf("Writing all metrics to %s", r.RemoteAddr))
		w.Write(out)
	}
}

func (wh *WebHandler) DispatchPOST(w http.ResponseWriter, r *http.Request) {
	req := wh.readAndUnmarshal(w, r, "POST")

	if req.Name != "" {
		out, err := json.Marshal(query.GetResult(req.Name))
		if err != nil {
			wh.Logger.Log("crit", fmt.Sprintf("Error gathering metrics for %s: %s", req.Name, err))
			w.WriteHeader(500)
		} else {
			wh.Logger.Log("debug", fmt.Sprintf("Handling POST for metric '%s'", req.Name))
			w.Write(out)
		}
	} else {
		wh.Logger.Log("debug", fmt.Sprintf("404ing because no payload from %s", r.RemoteAddr))
		w.WriteHeader(404)
	}
}

func (wh *WebHandler) DispatchPUT(w http.ResponseWriter, r *http.Request) {
	req := wh.readAndUnmarshal(w, r, "PUT")
	if req.Name == "" {
		wh.Logger.Log("crit", fmt.Sprintf("Cannot write record with an empty value"))
		w.WriteHeader(500)
	} else {
		record.RecordMetric(req.Name, req.Value, wh.Logger)
		wh.Logger.Log("debug", "here")
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	}
}

func (wh *WebHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if !wh.handleAuth(r) {
		wh.Logger.Log("info", fmt.Sprintf("Unauthorized access from %s", r.RemoteAddr))
		wh.showUnauthorized(w)
		return
	}

	httpMethodDispatch[r.Method](wh, w, r)
}

func Start(listen string, config types.CirconusConfig, log *logger.Logger) error {
	go query.ResultPoller(config, log)

	log.Log("info", "Starting Web Service")

	s := &http.Server{
		Addr:    listen,
		Handler: &WebHandler{Config: config, Logger: log},
	}

	return s.ListenAndServe()
}
