package handlers

import (
	"log"
	"net/http"

	"github.com/zenazn/goji/web"
)

// Root shows the index page of Red Skull, if you didn't want to display the
// dashboard at the root.
func Root(c web.C, w http.ResponseWriter, r *http.Request) {
	//ManagedConstellation.LoadPods()
	context := NewPageContext()
	context.Title = "Welcome to the Redis Manager"
	context.ViewTemplate = "index"
	log.Print("Index called")
	render(w, context)
}
