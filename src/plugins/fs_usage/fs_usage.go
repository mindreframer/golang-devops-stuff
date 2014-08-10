package fs_usage

import (
	"fmt"
	gm "github.com/gollector/gollector_metrics"
	"io/ioutil"
	"logger"
	"os"
	"regexp"
	"strings"
)

var SYSTEM_FILESYSTEMS = []string{
	"proc",
	"sysfs",
	"fusectl",
	"debugfs",
	"securityfs",
	"devtmpfs",
	"devpts",
	"tmpfs",
	"fuse",
}

const MTAB = "/etc/mtab"

const (
	PART_DISK       = 0
	PART_MOUNTPOINT = iota
	PART_FILESYSTEM = iota
	PART_FLAGS      = iota
	PART_DUMP       = iota
	PART_PASS       = iota
)

func Detect() []string {
	out, err := ioutil.ReadFile(MTAB)
	var collector []string

	if err != nil {
		fmt.Println("during detection, got error:", err)
		os.Exit(1)
	}

	lines := strings.Split(string(out), "\n")
	union := strings.Join(SYSTEM_FILESYSTEMS, "|")
	re, _ := regexp.Compile(union)

	split_re, _ := regexp.Compile("[ \t]+")

	for _, line := range lines {
		if line == "" {
			continue
		}

		parts := split_re.Split(line, -1)

		if re.Match([]byte(parts[PART_FILESYSTEM])) {
			continue
		} else {
			collector = append(collector, parts[PART_MOUNTPOINT])
		}
	}

	return collector
}

func GetMetric(params interface{}, log *logger.Logger) interface{} {
	info := gm.FSUsage(params.(string))

	return [5]interface{}{
		info.Free,
		info.Avail,
		info.Blocks,
		info.ReadOnly,
    info.Percent,
	}
}
