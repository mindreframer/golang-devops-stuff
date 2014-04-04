package ostential
import (
	"time"
	"net/url"
	"net/http"
	"github.com/gorilla/websocket"
)

func parseSearch(search string) (url.Values, error) {
	if search != "" && search[0] == '?' {
		search = search[1:]
	}
	return url.ParseQuery(search)
}

func init() {
	// collect() // collect for initial (first) `index' to have the data for display
	// although it may become outdated. TODO fix this:
	// 1. `index' should check for conntrack.count == 0, and if so, do collect() and reset timer

	go func() {
		for {
			select {
			case wc := <-register:
				if len(wclients) == 0 {
					collect() // for at least one new client
				}
				wclients[wc] = true

			case wc := <-unregister:
				delete(wclients, wc)
				close(wc.ping)
				if len(wclients) == 0 {
					reset_prev()
				}

			case <-time.After(time.Second * 1):
				collect()
				for wc := range wclients {
					wc.ping <- true
				}
			}
		}
	}()
}

type wclient struct {
	ws *websocket.Conn
	ping chan bool
	form url.Values
	new_search bool
}
var (
	 wclients  = make(map[ *wclient ]bool)
	  register = make(chan *wclient)
	unregister = make(chan *wclient)
)

func(wc *wclient) waitfor_messages() { // read from client
	defer wc.ws.Close()
	for {
		mt, data, err := wc.ws.ReadMessage()
		// websocket.Message.Receive(wc.ws, &search)
		if err != nil || mt != websocket.TextMessage {
			break
		}
		wc.form, err = parseSearch(string(data))
		if err != nil {
			// http.StatusBadRequest
			break
		}
		wc.new_search = true
		wc.ping <- true // don't wait for a second
	}
}
func(wc *wclient) waitfor_updates() { // write to  client
	defer func() {
		unregister <- wc
		wc.ws.Close()
	}()
	for {
		select {
		case <- wc.ping:
			send, _, _, _ := updates(&http.Request{Form: wc.form}, wc.new_search)
			wc.new_search = false

			if err := wc.ws.WriteJSON(send); err != nil {
				break
			}
		}
	}
}

func slashws(w http.ResponseWriter, req *http.Request) {
	if req.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if req.Header.Get("Origin") != "http://"+ req.Host {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	ws, err := websocket.Upgrade(w, req, nil, 1024, 1024)
	if err != nil {
		if _, ok := err.(websocket.HandshakeError); ok {
			http.Error(w, "websocket.Upgrade errd", http.StatusBadRequest)
			return
		}
		panic(err)
	}

	wc := &wclient{ws: ws, ping: make(chan bool, 1)}
	register <- wc
	defer func() {
		unregister <- wc
	}()
	go wc.waitfor_messages() // read from client
	   wc.waitfor_updates()  // write to  client
}
