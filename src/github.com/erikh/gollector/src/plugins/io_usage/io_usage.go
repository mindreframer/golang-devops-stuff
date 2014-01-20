package io_usage

import (
	"fmt"
	"io/ioutil"
	"logger"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
)

const DISKSTATS_FILE = "/proc/diskstats"

const (
	DEVICE_DISK uint = iota
	DEVICE_DM        = iota
)

const (
	LINE_ID           uint = 0
	LINE_DEVICE       uint = 2
	LINE_FIRST_METRIC uint = 3
)

var device_to_diskstat_id = map[uint]uint{
	DEVICE_DISK: 8,
	DEVICE_DM:   252,
}

var metric_names = []string{
	"reads issued",
	"reads merged",
	"sectors read",
	"time reading (ms)",
	"writes completed",
	"writes merged",
	"sectors written",
	"time writing (ms)",
	"iops in progress",
	"io time (ms)",
	"weighted io time (ms)",
}

var last_metrics map[string]map[string]uint64
var rwmutex sync.RWMutex

func getDeviceType(device_name string) uint {
	byte_dn := []byte(device_name)

	matched, _ := regexp.Match("^dm-", byte_dn)

	if matched {
		return device_to_diskstat_id[DEVICE_DM]
	}

	return device_to_diskstat_id[DEVICE_DISK]
}

func initLastMetrics(device string) (new_metrics bool) {
	new_metrics = false

	if last_metrics == nil {
		rwmutex.Lock()
		last_metrics = make(map[string]map[string]uint64)
		rwmutex.Unlock()
		new_metrics = true
	}

	if last_metrics[device] == nil {
		rwmutex.Lock()
		last_metrics[device] = make(map[string]uint64)
		rwmutex.Unlock()
		new_metrics = true
	}

	return new_metrics
}

func writeMetric(device string, metric string, value uint64) {
	rwmutex.Lock()
	last_metrics[device][metric] = value
	rwmutex.Unlock()
}

func readMetric(device string, metric string) (value uint64) {
	rwmutex.RLock()
	value = last_metrics[device][metric]
	rwmutex.RUnlock()

	return value
}

func getDiskMetrics(device string, device_type uint) (retval map[string]uint64, err error) {
	out, err := ioutil.ReadFile(DISKSTATS_FILE)

	if err != nil {
		return retval, err
	}

	lines := strings.Split(string(out), "\n")
	re, _ := regexp.Compile("[ \t]+")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := re.Split(line, -1)
		parts = parts[1:]

		device_type_parsed, err := strconv.ParseUint(parts[LINE_ID], 10, 64)

		if err != nil {
			return retval, err
		} else if uint(device_type_parsed) == device_type && parts[LINE_DEVICE] == device {
			retval = make(map[string]uint64)

			for i, key := range metric_names {
				retval[key], err = strconv.ParseUint(parts[LINE_FIRST_METRIC+uint(i)], 10, 64)

				if err != nil {
					fmt.Println(err)
					return make(map[string]uint64), err
				}
			}
		}
	}

	return retval, err
}

func GetMetric(params interface{}, log *logger.Logger) interface{} {

	difference := make(map[string]uint64)
	device := params.(string)
	device_type := getDeviceType(device)
	new_metrics := initLastMetrics(device)
	metrics, err := getDiskMetrics(device, device_type)

	if new_metrics {
		log.Log("debug", "New metrics, sending zeroes")
	}

	if err != nil {
		log.Log("crit", fmt.Sprintf("%s", err))
	}

	for metric, value := range metrics {
		if new_metrics {
			difference[metric] = 0
		} else {
			difference[metric] = value - readMetric(device, metric)
			if int64(value-readMetric(device, metric)) < 0 {
				difference[metric] = 0
			}
		}

		writeMetric(device, metric, value)
	}

	return difference
}

/* FIXME refactor to use getDiskMetrics's code for this in the future */

func Detect() []string {
	out, err := ioutil.ReadFile(DISKSTATS_FILE)
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

		device_type_parsed, err := strconv.ParseUint(parts[LINE_ID], 10, 64)

		if err != nil {
			fmt.Println("during detection, got error:", err)
			os.Exit(1)
		}

		/* FIXME
		     for some reason ram disks are detected as well -- figure out why
				 (or how to parse them)
		*/
		if uint(device_type_parsed) == device_to_diskstat_id[DEVICE_DISK] || uint(device_type_parsed) == device_to_diskstat_id[DEVICE_DM] {
			collector = append(collector, parts[LINE_DEVICE])
		}
	}

	return collector
}
