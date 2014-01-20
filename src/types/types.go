package types

import (
	"logger"
	"plugins/command"
	"plugins/cpu_usage"
	"plugins/fs_usage"
	"plugins/io_usage"
	"plugins/json_poll"
	"plugins/load_average"
	"plugins/mem_usage"
	"plugins/net_usage"
	"plugins/process_count"
	"plugins/process_mem_usage"
	"plugins/record"
	"plugins/socket_usage"
)

type PluginResult interface{}
type PluginResultCollection map[string]PluginResult

var Plugins = map[string]func(interface{}, *logger.Logger) interface{}{
	"load_average":      load_average.GetMetric,
	"cpu_usage":         cpu_usage.GetMetric,
	"mem_usage":         mem_usage.GetMetric,
	"command":           command.GetMetric,
	"net_usage":         net_usage.GetMetric,
	"io_usage":          io_usage.GetMetric,
	"record":            record.GetMetric,
	"fs_usage":          fs_usage.GetMetric,
	"json_poll":         json_poll.GetMetric,
	"socket_usage":      socket_usage.GetMetric,
	"process_count":     process_count.GetMetric,
	"process_mem_usage": process_mem_usage.GetMetric,
}

/*

How this works:

Key is the type of the metric, value is a func which returns the Params value
for any given metric.

The value + _ + the key is used to name the metric to allow for multiple
metrics of a single type.

*/

var Detectors = map[string]func() []string{
	"load_average": func() []string { return []string{} },
	"cpu_usage":    func() []string { return []string{} },
	"mem_usage":    func() []string { return []string{} },
	"net_usage":    net_usage.Detect,
	"io_usage":     io_usage.Detect,
	"fs_usage":     fs_usage.Detect,
}

type ConfigMap struct {
	Type   string
	Params interface{}
}

type PluginConfig map[string]ConfigMap

type CirconusConfig struct {
	Listen       string
	Username     string
	Password     string
	Facility     string
	LogLevel     string
	PollInterval uint
	Plugins      PluginConfig
}
