// +build !production

package main
import (
	"libostent"

	"net"
	"log"
	"flag"
	pprof "net/http/pprof"
)

func main() {
	flag.Parse()

	go ostent.Loop()
	// go ostent.CollectdLoop()

	listen, err := net.Listen("tcp", ostent.OstentBindFlag.String())
	if err != nil {
		log.Fatal(err)
	}
	log.Fatal(ostent.Serve(listen, false, ostent.Muxmap{
		"/debug/pprof/{name}":  pprof.Index,
		"/debug/pprof/cmdline": pprof.Cmdline,
		"/debug/pprof/profile": pprof.Profile,
		"/debug/pprof/symbol":  pprof.Symbol,
	}))
}




