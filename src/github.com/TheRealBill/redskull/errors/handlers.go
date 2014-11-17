package errors

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/therealbill/airbrake-go"
)

func init() {
	airbrake.ApiKey = os.Getenv("AIRBRAKE_API_KEY")
}

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

func throwSentinelConnectError(sentinel string, orig_err error, r *http.Request) {
	//em := fmt.Errorf("Sentinel '%s' Unavailable. Error=%s", sentinel, orig_err)
	e := airbrake.ExtendedNotification{ErrorClass: "Sentinel.Connection", Error: orig_err}
	err := airbrake.ExtendedError(e, r)
	if err != nil {
		log.Print("airbrake error:", err)
	}
}
