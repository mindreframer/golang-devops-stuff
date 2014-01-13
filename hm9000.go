package main

import (
	"fmt"
	"github.com/cloudfoundry/gosteno"

	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/hm"
	"github.com/codegangsta/cli"

	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "HM9000"
	app.Usage = "Start the various HM9000 components"
	app.Version = "0.0.9000"
	app.Commands = []cli.Command{
		{
			Name:        "fetch_desired",
			Description: "Fetches desired state",
			Usage:       "hm fetch_desired --config=/path/to/config --poll",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
				cli.BoolFlag{"poll", "If true, poll repeatedly with an interval defined in config"},
			},
			Action: func(c *cli.Context) {
				logger, _, conf := loadLoggerAndConfig(c, "fetcher")
				hm.FetchDesiredState(logger, conf, c.Bool("poll"))
			},
		},
		{
			Name:        "listen",
			Description: "Listens over the NATS for the actual state",
			Usage:       "hm listen --config=/path/to/config",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
			},
			Action: func(c *cli.Context) {
				logger, _, conf := loadLoggerAndConfig(c, "listener")
				hm.StartListeningForActual(logger, conf)
			},
		},
		{
			Name:        "analyze",
			Description: "Analyze the desired and actual state and enqueue start/stop messages",
			Usage:       "hm analyze --config=/path/to/config --poll",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
				cli.BoolFlag{"poll", "If true, poll repeatedly with an interval defined in config"},
			},
			Action: func(c *cli.Context) {
				logger, _, conf := loadLoggerAndConfig(c, "analyzer")
				hm.Analyze(logger, conf, c.Bool("poll"))
			},
		},
		{
			Name:        "send",
			Description: "Send the enqueued start/stop messages",
			Usage:       "hm send --config=/path/to/config --poll",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
				cli.BoolFlag{"poll", "If true, poll repeatedly with an interval defined in config"},
			},
			Action: func(c *cli.Context) {
				logger, _, conf := loadLoggerAndConfig(c, "sender")
				hm.Send(logger, conf, c.Bool("poll"))
			},
		},
		{
			Name:        "evacuator",
			Description: "Listens for Varz calls to serve metrics",
			Usage:       "hm evacuator --config=/path/to/config",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
			},
			Action: func(c *cli.Context) {
				logger, _, conf := loadLoggerAndConfig(c, "evacuator")
				hm.StartEvacuator(logger, conf)
			},
		},
		{
			Name:        "serve_metrics",
			Description: "Listens for Varz calls to serve metrics",
			Usage:       "hm serve_metrics --config=/path/to/config",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
			},
			Action: func(c *cli.Context) {
				logger, steno, conf := loadLoggerAndConfig(c, "metrics_server")
				hm.ServeMetrics(steno, logger, conf)
			},
		},
		{
			Name:        "serve_api",
			Description: "Serve app API over http",
			Usage:       "hm serve_api --config=/path/to/config",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
			},
			Action: func(c *cli.Context) {
				logger, _, conf := loadLoggerAndConfig(c, "api_server")
				hm.ServeAPI(logger, conf)
			},
		},
		{
			Name:        "shred",
			Description: "Deletes empty directories from the store",
			Usage:       "hm shred --config=/path/to/config --poll",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
				cli.BoolFlag{"poll", "If true, poll repeatedly with an interval defined in config"},
			},
			Action: func(c *cli.Context) {
				logger, _, conf := loadLoggerAndConfig(c, "shredder")
				hm.Shred(logger, conf, c.Bool("poll"))
			},
		},
		{
			Name:        "dump",
			Description: "Dumps contents of the data store",
			Usage:       "hm dump --config=/path/to/config",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
				cli.BoolFlag{"raw", "If set, dump the unstructured contents of the database"},
			},
			Action: func(c *cli.Context) {
				logger, _, conf := loadLoggerAndConfig(c, "dumper")
				hm.Dump(logger, conf, c.Bool("raw"))
			},
		},
	}

	app.Run(os.Args)
}

func loadLoggerAndConfig(c *cli.Context, component string) (logger.Logger, *gosteno.Logger, *config.Config) {
	configPath := c.String("config")
	if configPath == "" {
		fmt.Printf("Config path required")
		os.Exit(1)
	}

	conf, err := config.FromFile(configPath)
	if err != nil {
		fmt.Printf("Failed to load config: %s", err.Error())
		os.Exit(1)
	}

	stenoConf := &gosteno.Config{
		Sinks: []gosteno.Sink{
			gosteno.NewIOSink(os.Stdout),
			gosteno.NewSyslogSink("vcap.hm9000." + component),
		},
		Level: conf.LogLevel(),
		Codec: gosteno.NewJsonCodec(),
	}
	gosteno.Init(stenoConf)
	steno := gosteno.NewLogger("vcap.hm9000." + component)
	hmLogger := logger.NewRealLogger(steno)

	return hmLogger, steno, conf
}
