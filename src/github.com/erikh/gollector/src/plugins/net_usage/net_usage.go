package net_usage

import (
	"fmt"
	gm "github.com/gollector/gollector_metrics"
	"io/ioutil"
	"logger"
	"os"
)

var netUsage = &gm.NetUsage{}

func GetMetric(params interface{}, log *logger.Logger) interface{} {
	return netUsage.Metric(params.(string))
}

func Detect() []string {
	dirs, err := ioutil.ReadDir(gm.NET_BASE_PATH)

	var collector []string

	if err != nil {
		fmt.Println("during detection, got error:", err)
		os.Exit(1)
	}

	for _, dir := range dirs {
		collector = append(collector, dir.Name())
	}

	return collector
}
