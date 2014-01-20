package fs_usage

/*
// int statfs(const char *path, struct statfs *buf);

#include <sys/statvfs.h>
#include <stdlib.h>
#include <assert.h>

struct statvfs* go_statvfs(const char *path) {
  struct statvfs *fsinfo;
  fsinfo = malloc(sizeof(struct statvfs));
  assert(fsinfo != NULL);
  statvfs(path, fsinfo);
  return fsinfo;
}

int go_fs_readonly(const char *path) {
  struct statvfs *fsinfo = go_statvfs(path);

  return (fsinfo->f_flag & ST_RDONLY) == ST_RDONLY;
}
*/
import "C"

import (
	"fmt"
	"io/ioutil"
	"logger"
	"os"
	"regexp"
	"strings"
	"unsafe"
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
	path := C.CString(params.(string))
	stat := C.go_statvfs(path)
	readonly := C.go_fs_readonly(path)

	log.Log("debug", fmt.Sprintf("blocks size on %s: %v", string(*path), stat.f_bsize))
	log.Log("debug", fmt.Sprintf("blocks total on %s: %v", string(*path), stat.f_blocks))
	log.Log("debug", fmt.Sprintf("blocks free on %s: %v", string(*path), stat.f_bfree))

	defer C.free(unsafe.Pointer(stat))
	defer C.free(unsafe.Pointer(path))

	return [4]interface{}{
		(uint64(((float64(stat.f_blocks - stat.f_bfree)) / float64(stat.f_blocks)) * 100)),
		(uint64(stat.f_bfree) * uint64(stat.f_bsize)),
		(uint64(stat.f_blocks) * uint64(stat.f_bsize)),
		readonly == 1,
	}
}
