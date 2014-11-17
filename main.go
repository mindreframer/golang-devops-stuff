package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/kelseyhightower/envconfig"
	"github.com/therealbill/airbrake-go"
	"github.com/therealbill/redskull/actions"
	"github.com/therealbill/redskull/handlers"
	"github.com/zenazn/goji"
)

var Build string
var key string

type ConstellationConfig struct {
	Nodes           []string
	DeadNodes       []string
	ConnectionCount int64
}

func RefreshData() {
	t := time.Tick(60 * time.Second)
	for _ = range t {
		handlers.ManagedConstellation.LoadSentinelConfigFile()
		handlers.ManagedConstellation.GetAllSentinels()
		for _, pod := range handlers.ManagedConstellation.RemotePodMap {
			_, _ = handlers.ManagedConstellation.LocalPodMap[pod.Name]
			auth := handlers.ManagedConstellation.GetPodAuth(pod.Name)
			if pod.AuthToken != auth && auth > "" {
				pod.AuthToken = auth
			}
		}
		for _, pod := range handlers.ManagedConstellation.LocalPodMap {
			pod.AuthToken = handlers.ManagedConstellation.GetPodAuth(pod.Name)
		}
		handlers.ManagedConstellation.IsBalanced()
		log.Printf("Main Cache Stats: %+v", handlers.ManagedConstellation.AuthCache.GetStats())
		log.Printf("Hot Cache Stats: %+v", handlers.ManagedConstellation.AuthCache.GetHotStats())
	}
}

type LaunchConfig struct {
	Name                string
	Port                int
	IP                  string
	SentinelConfigFile  string
	GroupName           string
	BindAddress         string
	SentinelHostAddress string
	TemplateDirectory   string
}

var config LaunchConfig

func init() {
	err := envconfig.Process("redskull", &config)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Launch Config: %+v", config)
	if config.BindAddress > "" {
		flag.Set("bind", config.BindAddress)
	} else {
		if config.Port == 0 {
			log.Print("ENV contained no port, using default")
			config.Port = 8000
		}
	}
	ps := fmt.Sprintf("%s:%d", config.IP, config.Port)
	log.Printf("binding to '%s'", ps)
	flag.Set("bind", ps)

	if config.TemplateDirectory > "" {
		if !strings.HasSuffix(config.TemplateDirectory, "/") {
			config.TemplateDirectory += "/"
		}
	}
	handlers.TemplateBase = config.TemplateDirectory

	// handle absent sentinel config file w/a default
	if config.SentinelConfigFile == "" {
		log.Print("ENV contained no SentinelConfigFile, using default")
		config.SentinelConfigFile = "/etc/redis/sentinel.conf"
	}

	// handle absent sentinel config file w/a default
	if config.GroupName == "" {
		config.GroupName = "redskull:1"
		log.Print("ENV contained no GroupName, using default:" + config.GroupName)
	}

	key = os.Getenv("AIRBRAKE_API_KEY")
	airbrake.Endpoint = "https://api.airbrake.io/notifier_api/v2/notices"
	airbrake.ApiKey = key
	airbrake.Environment = os.Getenv("RSM_ENVIRONMENT")
	if len(airbrake.Environment) == 0 {
		airbrake.Environment = "Development"
	}
	if len(Build) == 0 {
		Build = ".1"
		return
	}
}

func main() {
	mc, err := actions.GetConstellation(config.Name, config.SentinelConfigFile, config.GroupName, config.SentinelHostAddress)
	if err != nil {
		log.Fatal("Unable to connect to constellation")
	}
	//log.Print("Starting refresh ticker")

	// Try to set to the IP the sentinel is bound do
	//flag.Set("bind", mc.SentinelConfig.Host+":8000")

	//go RefreshData()
	_, _ = mc.GetPodMap()
	//for _, pod := range pm {
	//handlers.NodeMaster.AddNode(pod.Master)
	//for _, node := range pod.Nodes { handlers.NodeMaster.AddNode(&node) }
	//}
	mc.IsBalanced()
	handlers.ManagedConstellation = mc
	_ = handlers.NewPageContext()
	if handlers.ManagedConstellation.AuthCache == nil {
		log.Print("Uninitialized AuthCache, StartCache not called, calling now")
		handlers.ManagedConstellation.StartCache()
	}
	log.Printf("Main Cache Stats: %+v", handlers.ManagedConstellation.AuthCache.GetStats())
	log.Printf("Hot Cache Stats: %+v", handlers.ManagedConstellation.AuthCache.GetHotStats())
	//log.Printf("MC:%+v", handlers.ManagedConstellation)

	// HTML Interface URLS
	goji.Get("/constellation/", handlers.ConstellationInfoHTML) // Needs moved? instance tree?
	goji.Get("/dashboard/", handlers.Dashboard)                 // Needs moved? instance tree?
	goji.Get("/constellation/addpodform/", handlers.AddPodForm)
	goji.Post("/constellation/addpod/", handlers.AddPodHTML)
	goji.Post("/constellation/addsentinel/", handlers.AddSentinelHTML)
	goji.Get("/constellation/addsentinelform/", handlers.AddSentinelForm)
	goji.Get("/constellation/rebalance/", handlers.RebalanceHTML)
	//goji.Get("/pod/:podName/dropslave", handlers.DropSlaveHTML)
	goji.Get("/pod/:podName/addslave", handlers.AddSlaveHTML)
	goji.Post("/pod/:podName/addslave", handlers.AddSlaveHTMLProcessor)
	goji.Post("/pod/:name/failover", handlers.DoFailoverHTML)
	goji.Post("/pod/:name/reset", handlers.ResetPodProcessor)
	goji.Post("/pod/:name/balance", handlers.BalancePodProcessor)
	goji.Get("/pod/:podName", handlers.ShowPod)
	goji.Get("/pods/", handlers.ShowPods)
	goji.Get("/nodes/", handlers.ShowNodes)
	goji.Get("/node/:name", handlers.ShowNode)
	goji.Get("/", handlers.Root) // Needs moved? instance tree?

	// API URLS
	goji.Get("/api/knownpods", handlers.APIGetPods)
	goji.Put("/api/monitor/:podName", handlers.APIMonitorPod)
	goji.Post("/api/constellation/:podName/failover", handlers.APIFailover)

	goji.Get("/api/pod/:podName", handlers.APIGetPod)
	goji.Put("/api/pod/:podName", handlers.APIMonitorPod)
	goji.Put("/api/pod/:podName/addslave", handlers.APIAddSlave)
	goji.Delete("/api/pod/:podName", handlers.APIRemovePod)
	goji.Get("/api/pod/:podName/master", handlers.APIGetMaster)
	goji.Get("/api/pod/:podName/slaves", handlers.APIGetSlaves)

	goji.Post("/api/node/clone", handlers.Clone) // Needs moved to the node tree
	goji.Get("/api/node/:name", handlers.GetNodeJSON)

	goji.Get("/static/*", handlers.Static) // Needs moved? instance tree?
	//goji.Abandon(middleware.Logger)

	goji.Serve()

}
