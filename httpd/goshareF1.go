package abkhttpd

import (
	"html/template"
	"net/http"
)

func Index(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	t, err := template.ParseFiles("httpd/public/index.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	t.Execute(w, nil)
}

func QuickStart(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	t, err := template.ParseFiles("httpd/public/quickstart.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	t.Execute(w, nil)
}

func HelpHTTP(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	t, err := template.ParseFiles("httpd/public/help-http.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	t.Execute(w, nil)
}

func HelpZMQ(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	t, err := template.ParseFiles("httpd/public/help-zmq.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	t.Execute(w, nil)
}

func Concept(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	t, err := template.ParseFiles("httpd/public/concept.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	t.Execute(w, nil)
}

func Status(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-Type", "text/html")

	t, err := template.ParseFiles("httpd/public/status.html")
	if err != nil {
		w.Write([]byte(err.Error()))
		return
	}
	t.Execute(w, nil)
}
