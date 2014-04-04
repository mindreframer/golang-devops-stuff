// +build !production

package main
import (
	"ostential"

	"os"
	"log"
	"flag"
	pprof "net/http/pprof"

	"github.com/codegangsta/martini"
)

func main() {
	martini.Env = martini.Dev

	listen, err := ostential.Listen()
	if err == nil {
		log.Fatal(ostential.Serve(listen, ostential.LogAll, func(m *ostential.Modern) {
			m.Any("/debug/pprof/cmdline", pprof.Cmdline)
			m.Any("/debug/pprof/profile", pprof.Profile)
			m.Any("/debug/pprof/symbol",  pprof.Symbol)
			m.Any("/debug/pprof/.*",      pprof.Index)
		}))
	}
	if _, ok := err.(ostential.FlagError); !ok {
		log.Fatal(err)
	}
	flag.Usage()
	os.Exit(2)
}




