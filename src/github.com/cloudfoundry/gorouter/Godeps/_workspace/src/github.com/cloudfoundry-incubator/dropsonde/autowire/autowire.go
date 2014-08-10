package autowire

import (
	"github.com/cloudfoundry-incubator/dropsonde"
	"github.com/cloudfoundry-incubator/dropsonde/emitter"
	"log"
	"net/http"
	"os"
)

var autowiredEmitter emitter.EventEmitter

var destination string

const defaultDestination = "localhost:42420"

func init() {
	Initialize()
}

func InstrumentedHandler(handler http.Handler) http.Handler {
	if autowiredEmitter == nil {
		return handler
	}

	return dropsonde.InstrumentedHandler(handler, autowiredEmitter)
}

func InstrumentedRoundTripper(roundTripper http.RoundTripper) http.RoundTripper {
	if autowiredEmitter == nil {
		return roundTripper
	}

	return dropsonde.InstrumentedRoundTripper(roundTripper, autowiredEmitter)
}

func Destination() string {
	return destination
}

func Initialize() {
	http.DefaultTransport = &http.Transport{Proxy: http.ProxyFromEnvironment}
	autowiredEmitter = nil

	origin := os.Getenv("DROPSONDE_ORIGIN")
	if len(origin) == 0 {
		log.Println("Failed to auto-initialize dropsonde: DROPSONDE_ORIGIN environment variable not set")
		return
	}

	destination = os.Getenv("DROPSONDE_DESTINATION")
	if len(destination) == 0 {
		log.Println("DROPSONDE_DESTINATION not set. Using " + defaultDestination)
		destination = defaultDestination
	}

	udpEmitter, err := emitter.NewUdpEmitter(destination)
	if err != nil {
		log.Printf("Failed to auto-initialize dropsonde: %v\n", err)
		return
	}

	hbEmitter, err := emitter.NewHeartbeatEmitter(udpEmitter, origin)
	if err != nil {
		log.Printf("Failed to auto-initialize dropsonde: %v\n", err)
		return
	}

	autowiredEmitter = emitter.NewEventEmitter(hbEmitter, origin)

	http.DefaultTransport = InstrumentedRoundTripper(http.DefaultTransport)
}
