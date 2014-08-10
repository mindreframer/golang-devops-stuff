package io_usage

import (
	"fmt"
	gm "github.com/gollector/gollector_metrics"
	"io/ioutil"
	"logger"
	"os"
	"regexp"
	"strconv"
	"strings"
)

var ioUsage = &gm.IOUsage{}

func GetMetric(params interface{}, log *logger.Logger) interface{} {
	results, err := ioUsage.Metric(params.(string))

	if err != nil {
		log.Log("crit", err.Error())
		return nil
	}

	return results
}

/* FIXME refactor to use getDiskMetrics's code for this in the future */

func Detect() []string {
	out, err := ioutil.ReadFile(gm.DISKSTATS_FILE)
	var collector []string

	if err != nil {
		fmt.Println("during detection, got error:", err)
		os.Exit(1)
	}

	lines := strings.Split(string(out), "\n")
	re, _ := regexp.Compile("[ \t]+")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := re.Split(line, -1)
		parts = parts[1:]

		device_type_parsed, err := strconv.ParseUint(parts[gm.LINE_ID], 10, 64)

		if err != nil {
			fmt.Println("during detection, got error:", err)
			os.Exit(1)
		}

		/* FIXME
		     for some reason ram disks are detected as well -- figure out why
				 (or how to parse them)
		*/
		if uint(device_type_parsed) == gm.DeviceMap[gm.DEVICE_DISK] || uint(device_type_parsed) == gm.DeviceMap[gm.DEVICE_DM] {
			collector = append(collector, parts[gm.LINE_DEVICE])
		}
	}

	return collector
}
