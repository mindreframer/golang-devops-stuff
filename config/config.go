package config

import (
	"github.com/cloudfoundry-incubator/candiedyaml"
	vcap "github.com/cloudfoundry/gorouter/common"

	"io/ioutil"
	"time"
)

type StatusConfig struct {
	Port uint16 `yaml:"port"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

var defaultStatusConfig = StatusConfig{
	Port: 8082,
	User: "",
	Pass: "",
}

type NatsConfig struct {
	Host string `yaml:"host"`
	Port uint16 `yaml:"port"`
	User string `yaml:"user"`
	Pass string `yaml:"pass"`
}

var defaultNatsConfig = NatsConfig{
	Host: "localhost",
	Port: 4222,
	User: "",
	Pass: "",
}

type LoggingConfig struct {
	File   string `yaml:"file"`
	Syslog string `yaml:"syslog"`
	Level  string `yaml:"level"`
}

var defaultLoggingConfig = LoggingConfig{
	Level: "debug",
}

type LoggregatorConfig struct {
	Url          string `yaml:"url"`
	SharedSecret string `yaml:"shared_secret"`
}

var defaultLoggregatorConfig = LoggregatorConfig{
	Url: "",
}

type Config struct {
	Status            StatusConfig      `yaml:"status"`
	Nats              []NatsConfig      `yaml:"nats"`
	Logging           LoggingConfig     `yaml:"logging"`
	LoggregatorConfig LoggregatorConfig `yaml:"loggregatorConfig"`

	Port       uint16 `yaml:"port"`
	Index      uint   `yaml:"index"`
	GoMaxProcs int    `yaml:"go_max_procs,omitempty"`
	TraceKey   string `yaml:"trace_key"`
	AccessLog  string `yaml:"access_log"`

	PublishStartMessageIntervalInSeconds int `yaml:"publish_start_message_interval"`
	PruneStaleDropletsIntervalInSeconds  int `yaml:"prune_stale_droplets_interval"`
	DropletStaleThresholdInSeconds       int `yaml:"droplet_stale_threshold"`
	PublishActiveAppsIntervalInSeconds   int `yaml:"publish_active_apps_interval"`
	StartResponseDelayIntervalInSeconds  int `yaml:"start_response_delay_interval"`
	EndpointTimeoutInSeconds             int `yaml:"endpoint_timeout"`
	DrainTimeoutInSeconds                int `yaml:"drain_timeout,omitempty"`

	// These fields are populated by the `Process` function.
	PruneStaleDropletsInterval time.Duration `yaml:"-"`
	DropletStaleThreshold      time.Duration `yaml:"-"`
	PublishActiveAppsInterval  time.Duration `yaml:"-"`
	StartResponseDelayInterval time.Duration `yaml:"-"`
	EndpointTimeout            time.Duration `yaml:"-"`
	DrainTimeout               time.Duration `yaml:"-"`

	Ip string `yaml:"-"`
}

var defaultConfig = Config{
	Status:            defaultStatusConfig,
	Nats:              []NatsConfig{defaultNatsConfig},
	Logging:           defaultLoggingConfig,
	LoggregatorConfig: defaultLoggregatorConfig,

	Port:       8081,
	Index:      0,
	GoMaxProcs: 8,

	EndpointTimeoutInSeconds: 60,

	PublishStartMessageIntervalInSeconds: 30,
	PruneStaleDropletsIntervalInSeconds:  30,
	DropletStaleThresholdInSeconds:       120,
	PublishActiveAppsIntervalInSeconds:   0,
	StartResponseDelayIntervalInSeconds:  5,
}

func DefaultConfig() *Config {
	c := defaultConfig

	c.Process()

	return &c
}

func (c *Config) Process() {
	var err error

	c.PruneStaleDropletsInterval = time.Duration(c.PruneStaleDropletsIntervalInSeconds) * time.Second
	c.DropletStaleThreshold = time.Duration(c.DropletStaleThresholdInSeconds) * time.Second
	c.PublishActiveAppsInterval = time.Duration(c.PublishActiveAppsIntervalInSeconds) * time.Second
	c.StartResponseDelayInterval = time.Duration(c.StartResponseDelayIntervalInSeconds) * time.Second
	c.EndpointTimeout = time.Duration(c.EndpointTimeoutInSeconds) * time.Second

	drain := c.DrainTimeoutInSeconds
	if drain == 0 {
		drain = c.EndpointTimeoutInSeconds
	}
	c.DrainTimeout = time.Duration(drain) * time.Second

	c.Ip, err = vcap.LocalIP()
	if err != nil {
		panic(err)
	}
}

func (c *Config) Initialize(configYAML []byte) error {
	c.Nats = []NatsConfig{}
	return candiedyaml.Unmarshal(configYAML, &c)
}

func InitConfigFromFile(path string) *Config {
	var c *Config = DefaultConfig()
	var e error

	b, e := ioutil.ReadFile(path)
	if e != nil {
		panic(e.Error())
	}

	e = c.Initialize(b)
	if e != nil {
		panic(e.Error())
	}

	c.Process()

	return c
}
