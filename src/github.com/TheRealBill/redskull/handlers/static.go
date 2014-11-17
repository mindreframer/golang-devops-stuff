package handlers

import (
	"io"
	"log"
	"net/http"
	"time"

	"github.com/zenazn/goji/web"
)

const STATIC_URL string = "http://redskull.iamtherealbill.com/static/"
const STATIC_ROOT string = "html/static/"

// Static serves static content such as images, JS and CSS files
func Static(c web.C, w http.ResponseWriter, req *http.Request) {
	//log.Printf("STATIC_URL: %s, %d", STATIC_URL, len(STATIC_URL))
	//log.Printf("PATH: %s, %d", req.URL.Path, len(STATIC_URL))
	static_file := req.URL.Path[7:]
	root := TemplateBase + STATIC_ROOT
	if len(static_file) != 0 {
		f, err := http.Dir(root).Open(static_file)
		defer f.Close()
		if err == nil {
			content := io.ReadSeeker(f)
			http.ServeContent(w, req, static_file, time.Now(), content)
			return
		}
		log.Print(err)
	}
	http.NotFound(w, req)
}
