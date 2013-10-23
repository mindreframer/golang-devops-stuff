package main

import (
	"flag"

	"github.com/cloudfoundry/gorouter"
	"github.com/cloudfoundry/gorouter/config"
	"github.com/cloudfoundry/gorouter/log"
)

var configFile string

func init() {
	flag.StringVar(&configFile, "c", "", "Configuration File")

	flag.Parse()
}

func main() {
	c := config.DefaultConfig()
	if configFile != "" {
		c = config.InitConfigFromFile(configFile)
	}

	log.SetupLoggerFromConfig(c)

	router.NewRouter(c).Run()

	select {}
}
