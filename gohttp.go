package goshare

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"

	golhashmap "github.com/abhishekkr/gol/golhashmap"
	"github.com/abhishekkr/gol/goltime"
	"github.com/abhishekkr/goshare/httpd"
)

/* enabling HTTP to formulate DBTasks call from HTTP Request */
func DBRest(httpMethod string, w http.ResponseWriter, req *http.Request) {
	var (
		dbAction       string
		response_bytes []byte
		axn_status     bool
	)

	switch httpMethod {
	case "GET":
		dbAction = "read"

	case "POST", "PUT":
		dbAction = "push"

	case "DELETE":
		dbAction = "delete"

	default:
		// log_this corrupt request
		return
	}

	packet := PacketFromHTTPRequest(dbAction, req)
	if packet.DBAction != "" {
		response_bytes, axn_status = DBTasksOnPacket(packet)
		DBRestResponse(w, req, response_bytes, axn_status)
	}
}

/* send proper response back to client based on success/data/error */
func DBRestResponse(w http.ResponseWriter, req *http.Request, response_bytes []byte, axn_status bool) {
	if !axn_status {
		error_msg := fmt.Sprintf("FATAL Error: (DBTasks) %q", req.Form)
		http.Error(w, error_msg, http.StatusInternalServerError)

	} else if len(response_bytes) == 0 {
		w.Write([]byte("Success"))

	} else {
		w.Write(response_bytes)

	}
}

/*
return Packet identifiable by DBTasksOnAction
*/
func PacketFromHTTPRequest(dbAction string, req *http.Request) Packet {
	packet := Packet{}
	packet.HashMap = make(golhashmap.HashMap)
	packet.DBAction = dbAction

	req.ParseForm()
	task_type := req.FormValue("type")
	if task_type == "" {
		task_type = "default"
	}
	packet.TaskType = task_type
	task_type_tokens := strings.Split(task_type, "-")
	packet.KeyType = task_type_tokens[0]
	if packet.KeyType == "tsds" && packet.DBAction == "push" {
		packet.TimeDot = goltime.TimestampFromHTTPRequest(req)
	}

	if len(task_type_tokens) > 1 {
		packet.ValType = task_type_tokens[1]

		if len(task_type_tokens) == 3 {
			thirdTokenFeatureHTTP(&packet, req)
		}
	}

	dbdata := req.FormValue("dbdata")
	key := req.FormValue("key")
	if key != "" {
		dbdata = fmt.Sprintf("%s\n%s", key, req.FormValue("val"))
	} else if dbdata == "" {
		return Packet{}
	}
	decodeData(&packet, strings.Split(dbdata, "\n"))

	return packet
}

/* third token handler in taskType */
func thirdTokenFeatureHTTP(packet *Packet, req *http.Request) {
	parentNS := req.FormValue("parentNS")
	if parentNS != "" {
		packet.ParentNS = parentNS
	}
}

/* DB Call HTTP Handler */
func DBRestHandler(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	req.ParseForm()

	DBRest(req.Method, w, req)
}

/* HTTP GET DB-GET call handler */
func GetReadKey(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	req.ParseForm()

	DBRest("GET", w, req)
}

/* HTTP GET DB-POST call handler */
func GetPushKey(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	req.ParseForm()

	DBRest("POST", w, req)
}

/* HTTP GET DB-POST call handler */
func GetDeleteKey(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	req.ParseForm()

	DBRest("DELETE", w, req)
}

/*
GoShare Handler for HTTP Requests
*/
func GoShareHTTP(httpuri string, httpport int) {
	runtime.GOMAXPROCS(runtime.NumCPU())

	http.HandleFunc("/", abkhttpd.Index)
	http.HandleFunc("/quickstart", abkhttpd.QuickStart)
	http.HandleFunc("/help-http", abkhttpd.HelpHTTP)
	http.HandleFunc("/help-zmq", abkhttpd.HelpZMQ)
	http.HandleFunc("/concept", abkhttpd.Concept)
	http.HandleFunc("/status", abkhttpd.Status)

	http.HandleFunc("/db", DBRestHandler)
	http.HandleFunc("/get", GetReadKey)
	http.HandleFunc("/put", GetPushKey)
	http.HandleFunc("/del", GetDeleteKey)

	srv := &http.Server{
		Addr:        fmt.Sprintf("%s:%d", httpuri, httpport),
		Handler:     http.DefaultServeMux,
		ReadTimeout: time.Duration(5) * time.Second,
	}

	fmt.Printf("access your goshare at http://%s:%d\n", httpuri, httpport)
	err := srv.ListenAndServe()
	fmt.Println("Game Over:", err)
}
