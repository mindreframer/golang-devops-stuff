package main

import (
	"github.com/vektra/tachyon"
	_ "github.com/vektra/tachyon/net"
	_ "github.com/vektra/tachyon/package"
	_ "github.com/vektra/tachyon/procmgmt"
	"os"
)

var Release string

func main() {
	if Release != "" {
		tachyon.Release = Release
	}

	os.Exit(tachyon.Main(os.Args))
}
