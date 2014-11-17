package handlers

import (
	"fmt"
	"log"
	"net/http"

	"github.com/zenazn/goji/web"
)

// DoFailoverHTML is how the UI initiates a failover for a pod
func DoFailoverHTML(c web.C, w http.ResponseWriter, r *http.Request) {
	// Needs changed to use templates!
	podname := c.URLParams["name"]
	pc := NewPageContext()
	pc.ViewTemplate = "failover-requested"
	pc.Refresh = true
	pc.RefreshTime = 10
	pc.RefreshURL = fmt.Sprintf("/pod/%s", podname)
	log.Printf("Failover requested for pod '%s'", podname)
	didFailover, err := ManagedConstellation.Failover(podname)
	if err != nil {
		retcode, emsg := handleFailoverError(podname, r, err)
		log.Printf("%d: '%s'", retcode, emsg)
	}
	if !didFailover {
		retcode, emsg := handleFailoverError(podname, r, err)
		log.Printf("%d: '%s'", retcode, emsg)
	}
	render(w, pc)
}
