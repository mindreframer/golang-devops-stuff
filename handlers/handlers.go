// Package handlers is where the HTTP server work is done.
package handlers

// TODO: Move error handlers to error package

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"text/template"

	"github.com/dustin/go-humanize"
	"github.com/therealbill/airbrake-go"
	"github.com/therealbill/redskull/actions"
)

// ManagedConstellation represents the constellation serveed by this Red Skull
// instance This will need refactored to use the new termin for the
// super-constellation reflecting the evoluton of the constellation term
var ManagedConstellation *actions.Constellation

// NodeMaster is deprecated. Previously/currently used for storing node
// connections. It needs refactored to use the constellation-wide node routines
var NodeMaster = new(actions.NodeStore)

var TemplateBase string

// InfoResponse represents the information returned in an API call
type InfoResponse struct {
	Status        string
	StatusMessage string
	Data          interface{}
}

// PageContext holds all the contextual information a page will want to return,
// use, or display
type PageContext struct {
	Title         string
	SubTitle      string
	Data          interface{}
	Static        string
	ViewTemplate  string
	CurrentURL    string
	Constellation *actions.Constellation
	NodeMaster    actions.NodeManager
	Pod           *actions.RedisPod
	Node          *actions.RedisNode
	Refresh       bool
	RefreshTime   int
	RefreshURL    string
	Error         error
}

// NewPageContext instantiates and returns a PageContext with "global" data
// already set.
func NewPageContext() (pc PageContext) {
	pc = PageContext{Static: STATIC_URL, Constellation: ManagedConstellation, NodeMaster: NodeMaster}
	return
}

// getTemplateList returns the base template and the requested template
func getTemplateList(tname string) []string {
	base := TemplateBase + "html/templates/base.html"
	thisOne := TemplateBase + "html/templates/" + tname + ".html"
	tmpl_list := []string{base, thisOne}
	return tmpl_list
}

// HumanizeBigBytes transforms a uint64 to a human readable string such as
// "100Kb"
func HumanizeBigBytes(bytes int64) string {
	res := humanize.Bytes(uint64(bytes))
	return res
}

// HumanizeBytes transforms an int to a human readable string such as
// "100Kb"
func HumanizeBytes(bytes int) string {
	res := humanize.Bytes(uint64(bytes))
	return res
}

// CommifyFloat turns a float into a string with comma separation
func CommifyFloat(bytes float64) string {
	res := humanize.Comma(int64(bytes))
	return res
}

// HumanizeSlowlog is used to convert Redis' calls stats to something humans
// can easily read. This means converting microseconds to milliseconds where
// appropriate, adding commas, and hopefully soon upconverting to seconds,
// minutes, hours, days, etc. where approriate
func HumanizeSlowlog(micros int64) (res string) {
	nt := micros / 1000.0
	if nt > 1 {
		res = fmt.Sprintf("%s milliseconds", humanize.Comma(int64(nt)))
	} else {
		res = fmt.Sprintf("%d microseconds", micros)
	}
	return res
}

// HumanizeCallStats is used to convert Redis' calls stats to something humans
// can easily read. This means converting microseconds to milliseconds where
// appropriate, adding commas, and hopefully soon upconverting to seconds,
// minutes, hours, days, etc. where approriate
func HumanizeCallStats(micros float64) (res string) {
	nt := micros / 1000.0
	if nt > 1 {
		res = fmt.Sprintf("%s milliseconds", humanize.Comma(int64(nt)))
	} else {
		res = fmt.Sprintf("%s microseconds", humanize.Ftoa(micros))
	}
	return res
}

// IntFromFloat64 provides a convenience function fo convert an int to a float
// insert screed about how you probably should not do it but sometimes you need
// to here.
func IntFromFloat64(incoming float64) (i int) {
	i = int(incoming)
	return i
}

// Turn an "ok" string into a boolean
func OkToBool(ok string) bool {
	if ok == "ok" {
		return true
	}
	return false
}

// render is called to turn the processed data into a rendered page in a
// handelr function. This way we can add features such as authentication to a
// central rendering call rather than go through each web page function and do
// it there.
func render(w http.ResponseWriter, context PageContext) {
	funcMap := template.FuncMap{
		"title":             strings.Title,
		"HumanizeBytes":     HumanizeBytes,
		"HumanizeBigBytes":  HumanizeBigBytes,
		"CommifyFloat":      CommifyFloat,
		"Float2Int":         IntFromFloat64,
		"OkToBool":          OkToBool,
		"HumanizeCallStats": HumanizeCallStats,
		"HumanizeSlowlog":   HumanizeSlowlog,
		"tableflip":         func() string { return "(╯°□°）╯︵ ┻━┻" },
	}
	context.Static = STATIC_URL
	tmpl_list := getTemplateList(context.ViewTemplate)
	/*
		t, err := template.ParseFiles(tmpl_list...)
		if err != nil {
			log.Print("template parsing error: ", err)
		}
	*/
	t := template.Must(template.New("base.html").Funcs(funcMap).ParseFiles(tmpl_list...))
	err := t.Execute(w, context)
	if err != nil {
		log.Print("template executing error: ", err)
	}
}

// currently only used to set up your airbrake key
func init() {
	airbrake.ApiKey = os.Getenv("AIRBRAKE_API_KEY")
}

// throwJSONParseError is used when the JSON a client submits via the API isn't
// parseable
func throwJSONParseError(req *http.Request) (retcode int, userMessage string) {
	retcode = 422
	userMessage = "JSON Parse failure"
	em := fmt.Errorf(userMessage)
	e := airbrake.ExtendedNotification{ErrorClass: "Request.ParseJSON", Error: em}
	err := airbrake.ExtendedError(e, req)
	if err != nil {
		log.Print("airbrake error:", err)
	}
	return
}

// handleFailoverError is/wa sused to handle sentinel being unable to failover.
// This is going to need to be used to track times when a call initiated a
// failover that failed.
func handleFailoverError(pod string, req *http.Request, orig_err error) (retcode int, userMessage string) {
	var em error
	retcode = 500
	if strings.Contains(orig_err.Error(), "No such master with that name") {
		userMessage = "No pod or master with that name was found"
		log.Printf("Failover request for nonexistent pod: '%s'", pod)
		retcode = http.StatusNotFound
		return
	}
	if strings.Contains(orig_err.Error(), "INPROG") {
		userMessage = "Enhance your calm. Failover is in progress"
		log.Printf("Attempt to failover pod '%s' during failover", pod)
		//em = fmt.Errorf("Failover Error: podName='%s', err='%s'", pod, userMessage)
		retcode = 420
		return
	}
	e := airbrake.ExtendedNotification{ErrorClass: "Sentinel.Failover", Error: em}
	err := airbrake.ExtendedError(e, req)
	if err != nil {
		log.Print("airbrake error:", err)
	}
	userMessage = em.Error()
	return
}

// throwSentinelConnectError is/was used for failed attempts to connect to a
// sentinel.
func throwSentinelConnectError(sentinel string, orig_err error, r *http.Request) {
	//em := fmt.Errorf("Sentinel '%s' Unavailable. Error=%s", sentinel, orig_err)
	e := airbrake.ExtendedNotification{ErrorClass: "Sentinel.Connection", Error: orig_err}
	err := airbrake.ExtendedError(e, r)
	if err != nil {
		log.Print("airbrake error:", err)
	}
}
