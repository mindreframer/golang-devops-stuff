package mem_usage

import (
	"fmt"
	"io/ioutil"
	"logger"
	"strconv"
	"strings"
)

func GetMetric(params interface{}, log *logger.Logger) interface{} {
	log.Log("debug", "Reading /proc/meminfo")
	content, err := ioutil.ReadFile("/proc/meminfo")

	var total, buffers, cached, free, swap_total, swap_free int

	if err != nil {
		log.Log("crit", fmt.Sprintf("While processing the mem_usage package: %s", err))
		return map[string]interface{}{}
	}

	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		parts := strings.Split(line, " ")
		id := len(parts) - 2

		switch parts[0] {
		case "MemTotal:":
			total, err = strconv.Atoi(parts[id])
		case "MemFree:":
			free, err = strconv.Atoi(parts[id])
		case "Cached:":
			cached, err = strconv.Atoi(parts[id])
		case "Buffers:":
			buffers, err = strconv.Atoi(parts[id])
		case "SwapTotal:":
			swap_total, err = strconv.Atoi(parts[id])
		case "SwapFree:":
			swap_free, err = strconv.Atoi(parts[id])
		}

		if err != nil {
			log.Log("crit", fmt.Sprintf("Could not convert integer from string while processing cpu_usage: %s", parts[id]))
			return map[string]interface{}{}
		}
	}

	return map[string]interface{}{
		"Total":      total * 1024,
		"Free":       (buffers + cached + free) * 1024,
		"Used":       total*1024 - ((buffers + cached + free) * 1024),
		"Swap Total": swap_total * 1024,
		"Swap Free":  swap_free * 1024,
	}
}
