package main

import (
	"github.com/cloudfoundry/gosteno"

	"github.com/cloudfoundry/hm9000/config"
	"github.com/cloudfoundry/hm9000/helpers/logger"
	"github.com/cloudfoundry/hm9000/hm"
	"github.com/codegangsta/cli"

	"os"
)

func main() {
	c := &gosteno.Config{
		Sinks: []gosteno.Sink{
			gosteno.NewSyslogSink("hm9000"),
		},
		Level:     gosteno.LOG_INFO,
		Codec:     gosteno.NewJsonCodec(),
		EnableLOC: true,
	}
	gosteno.Init(c)
	steno := gosteno.NewLogger("hm9000")
	l := logger.NewRealLogger(steno)

	app := cli.NewApp()
	app.Name = "HM9000"
	app.Usage = "Start the various HM9000 components"
	app.Version = "0.0.9000"
	app.Commands = []cli.Command{
		cli.Command{
			Name:        "fetch_desired",
			Description: "Fetches desired state",
			Usage:       "hm fetch_desired --config=/path/to/config --poll",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
				cli.BoolFlag{"poll", "If true, poll repeatedly with an interval defined in config"},
			},
			Action: func(c *cli.Context) {
				hm.FetchDesiredState(l, loadConfig(l, c), c.Bool("poll"))
			},
		},
		cli.Command{
			Name:        "listen",
			Description: "Listens over the NATS for the actual state",
			Usage:       "hm listen --config=/path/to/config",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
			},
			Action: func(c *cli.Context) {
				hm.StartListeningForActual(l, loadConfig(l, c))
			},
		},
		cli.Command{
			Name:        "analyze",
			Description: "Analyze the desired and actual state and enqueue start/stop messages",
			Usage:       "hm analyze --config=/path/to/config --poll",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
				cli.BoolFlag{"poll", "If true, poll repeatedly with an interval defined in config"},
			},
			Action: func(c *cli.Context) {
				hm.Analyze(l, loadConfig(l, c), c.Bool("poll"))
			},
		},
		cli.Command{
			Name:        "send",
			Description: "Send the enqueued start/stop messages",
			Usage:       "hm send --config=/path/to/config --poll",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
				cli.BoolFlag{"poll", "If true, poll repeatedly with an interval defined in config"},
			},
			Action: func(c *cli.Context) {
				hm.Send(l, loadConfig(l, c), c.Bool("poll"))
			},
		},
		cli.Command{
			Name:        "serve_metrics",
			Description: "Listens for Varz calls to serve metrics",
			Usage:       "hm serve_metrics --config=/path/to/config",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
			},
			Action: func(c *cli.Context) {
				hm.ServeMetrics(steno, l, loadConfig(l, c))
			},
		},
		cli.Command{
			Name:        "serve_api",
			Description: "Serve app API over http",
			Usage:       "hm serve_api --config=/path/to/config",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
			},
			Action: func(c *cli.Context) {
				hm.ServeAPI(l, loadConfig(l, c))
			},
		},
		cli.Command{
			Name:        "dump",
			Description: "Dumps contents of the data store",
			Usage:       "hm dump --config=/path/to/config",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
			},
			Action: func(c *cli.Context) {
				hm.Dump(l, loadConfig(l, c))
			},
		},
		cli.Command{
			Name:        "clear_store",
			Description: "Clears contents of the data store",
			Usage:       "hm clear_store --config=/path/to/config",
			Flags: []cli.Flag{
				cli.StringFlag{"config", "", "Path to config file"},
			},
			Action: func(c *cli.Context) {
				hm.Clear(l, loadConfig(l, c))
			},
		},
	}

	app.Run(os.Args)
}

func loadConfig(l logger.Logger, c *cli.Context) config.Config {
	configPath := c.String("config")
	if configPath == "" {
		l.Info("Config path required", nil)
		os.Exit(1)
	}

	conf, err := config.FromFile(configPath)
	if err != nil {
		l.Info("Failed to load config", map[string]string{"Error": err.Error()})
		os.Exit(1)
	}

	return conf
}
