package handlers

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/therealbill/libredis/client"
	"github.com/zenazn/goji/web"
)

// Info is deprecated in favor of the cluster level node routines
func Info(c web.C, w http.ResponseWriter, r *http.Request) {
	var response InfoResponse
	target := c.URLParams["targetAddress"]
	section := c.URLParams["section"]
	_ = section
	conn, err := client.DialWithConfig(&client.DialConfig{Address: target})

	if err != nil {
		response.Status = "CONNECTIONERROR"
		response.StatusMessage = "Unable to connect to specified Redis instance"
		fmt.Fprint(w, response)
	} else {
		defer conn.ClosePool()
		info, err := conn.Info()
		if err != nil {
			response.Status = "COMMANDERROR"
			response.StatusMessage = err.Error()
		} else {

			response.Data = info
			response.Status = "SUCCESS"
		}
	}
	packed, err := json.Marshal(response)
	if err != nil {
		log.Printf("JSON Marshalling Error: %s", err)
	}
	w.Write(packed)
}
