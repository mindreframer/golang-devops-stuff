package config

import (
	"encoding/json"
	"github.com/cloudfoundry/gosteno"
	"io/ioutil"
	"path/filepath"
	"runtime"
	"time"
)

type Config struct {
	HeartbeatPeriod                 uint64 `json:"heartbeat_period_in_seconds"`
	HeartbeatTTLInHeartbeats        uint64 `json:"heartbeat_ttl_in_heartbeats"`
	ActualFreshnessTTLInHeartbeats  uint64 `json:"actual_freshness_ttl_in_heartbeats"`
	GracePeriodInHeartbeats         int    `json:"grace_period_in_heartbeats"`
	DesiredFreshnessTTLInHeartbeats uint64 `json:"desired_freshness_ttl_in_heartbeats"`

	SenderPollingIntervalInHeartbeats   int `json:"sender_polling_interval_in_heartbeats"`
	SenderTimeoutInHeartbeats           int `json:"sender_timeout_in_heartbeats"`
	FetcherPollingIntervalInHeartbeats  int `json:"fetcher_polling_interval_in_heartbeats"`
	FetcherTimeoutInHeartbeats          int `json:"fetcher_timeout_in_heartbeats"`
	ShredderPollingIntervalInHeartbeats int `json:"shredder_polling_interval_in_heartbeats"`
	ShredderTimeoutInHeartbeats         int `json:"shredder_timeout_in_heartbeats"`
	AnalyzerPollingIntervalInHeartbeats int `json:"analyzer_polling_interval_in_heartbeats"`
	AnalyzerTimeoutInHeartbeats         int `json:"analyzer_timeout_in_heartbeats"`

	ListenerHeartbeatSyncIntervalInMilliseconds      int `json:"listener_heartbeat_sync_interval_in_milliseconds"`
	StoreHeartbeatCacheRefreshIntervalInMilliseconds int `json:"store_heartbeat_cache_refresh_interval_in_milliseconds"`

	DesiredStateBatchSize          int    `json:"desired_state_batch_size"`
	FetcherNetworkTimeoutInSeconds int    `json:"fetcher_network_timeout_in_seconds"`
	ActualFreshnessKey             string `json:"actual_freshness_key"`
	DesiredFreshnessKey            string `json:"desired_freshness_key"`
	CCAuthUser                     string `json:"cc_auth_user"`
	CCAuthPassword                 string `json:"cc_auth_password"`
	CCBaseURL                      string `json:"cc_base_url"`

	StoreSchemaVersion         int      `json:"store_schema_version"`
	StoreType                  string   `json:"store_type"`
	StoreURLs                  []string `json:"store_urls"`
	StoreMaxConcurrentRequests int      `json:"store_max_concurrent_requests"`

	SenderNatsStartSubject string `json:"sender_nats_start_subject"`
	SenderNatsStopSubject  string `json:"sender_nats_stop_subject"`
	SenderMessageLimit     int    `json:"sender_message_limit"`

	NumberOfCrashesBeforeBackoffBegins int `json:"number_of_crashes_before_backoff_begins"`
	StartingBackoffDelayInHeartbeats   int `json:"starting_backoff_delay_in_heartbeats"`
	MaximumBackoffDelayInHeartbeats    int `json:"maximum_backoff_delay_in_heartbeats"`

	MetricsServerPort     int    `json:"metrics_server_port"`
	MetricsServerUser     string `json:"metrics_server_user"`
	MetricsServerPassword string `json:"metrics_server_password"`

	LogLevelString string `json:"log_level"`

	NATS []struct {
		Host     string `json:"host"`
		Port     int    `json:"port"`
		User     string `json:"user"`
		Password string `json:"password"`
	} `json:"nats"`
}

func (conf *Config) HeartbeatTTL() uint64 {
	return conf.HeartbeatTTLInHeartbeats * conf.HeartbeatPeriod
}

func (conf *Config) ActualFreshnessTTL() uint64 {
	return conf.ActualFreshnessTTLInHeartbeats * conf.HeartbeatPeriod
}

func (conf *Config) GracePeriod() int {
	return conf.GracePeriodInHeartbeats * int(conf.HeartbeatPeriod)
}

func (conf *Config) DesiredFreshnessTTL() uint64 {
	return conf.DesiredFreshnessTTLInHeartbeats * conf.HeartbeatPeriod
}

func (conf *Config) FetcherNetworkTimeout() time.Duration {
	return time.Duration(conf.FetcherNetworkTimeoutInSeconds) * time.Second
}

func (conf *Config) SenderPollingInterval() time.Duration {
	return time.Duration(conf.SenderPollingIntervalInHeartbeats*int(conf.HeartbeatPeriod)) * time.Second
}

func (conf *Config) SenderTimeout() time.Duration {
	return time.Duration(conf.SenderTimeoutInHeartbeats*int(conf.HeartbeatPeriod)) * time.Second
}

func (conf *Config) FetcherPollingInterval() time.Duration {
	return time.Duration(conf.FetcherPollingIntervalInHeartbeats*int(conf.HeartbeatPeriod)) * time.Second
}

func (conf *Config) FetcherTimeout() time.Duration {
	return time.Duration(conf.FetcherTimeoutInHeartbeats*int(conf.HeartbeatPeriod)) * time.Second
}

func (conf *Config) ShredderPollingInterval() time.Duration {
	return time.Duration(conf.ShredderPollingIntervalInHeartbeats*int(conf.HeartbeatPeriod)) * time.Second
}

func (conf *Config) ShredderTimeout() time.Duration {
	return time.Duration(conf.ShredderTimeoutInHeartbeats*int(conf.HeartbeatPeriod)) * time.Second
}

func (conf *Config) AnalyzerPollingInterval() time.Duration {
	return time.Duration(conf.AnalyzerPollingIntervalInHeartbeats*int(conf.HeartbeatPeriod)) * time.Second
}

func (conf *Config) AnalyzerTimeout() time.Duration {
	return time.Duration(conf.AnalyzerTimeoutInHeartbeats*int(conf.HeartbeatPeriod)) * time.Second
}

func (conf *Config) StartingBackoffDelay() time.Duration {
	return time.Duration(conf.StartingBackoffDelayInHeartbeats*int(conf.HeartbeatPeriod)) * time.Second
}

func (conf *Config) MaximumBackoffDelay() time.Duration {
	return time.Duration(conf.MaximumBackoffDelayInHeartbeats*int(conf.HeartbeatPeriod)) * time.Second
}

func (conf *Config) ListenerHeartbeatSyncInterval() time.Duration {
	return time.Millisecond * time.Duration(conf.ListenerHeartbeatSyncIntervalInMilliseconds)
}

func (conf *Config) StoreHeartbeatCacheRefreshInterval() time.Duration {
	return time.Millisecond * time.Duration(conf.StoreHeartbeatCacheRefreshIntervalInMilliseconds)
}

func (conf *Config) LogLevel() gosteno.LogLevel {
	switch conf.LogLevelString {
	case "INFO":
		return gosteno.LOG_INFO
	case "DEBUG":
		return gosteno.LOG_DEBUG
	default:
		return gosteno.LOG_INFO
	}
}

func DefaultConfig() (*Config, error) {
	_, file, _, _ := runtime.Caller(0)
	pathToJSON := filepath.Clean(filepath.Join(filepath.Dir(file), "default_config.json"))

	return FromFile(pathToJSON)
}

func FromFile(path string) (*Config, error) {
	json, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
	}

	return FromJSON(json)
}

func FromJSON(JSON []byte) (*Config, error) {
	var config Config
	err := json.Unmarshal(JSON, &config)
	if err == nil {
		return &config, nil
	} else {
		return nil, err
	}
}
