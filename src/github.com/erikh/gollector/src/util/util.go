package util

import (
	"io/ioutil"
	"logger"
	"os"
	"strconv"
	"strings"
)

/*
 * Get the process ids for a given process name, full path is required.
 *
 * Returns strings because it's easier for most of the things we'll use this
 * for.
 */

func GetPids(process string, log *logger.Logger) []string {
	pids := []string{}

	dir, err := os.Open("/proc")

	if err != nil {
		log.Log("crit", "Could not open /proc for reading: "+err.Error())
		return nil
	}

	defer dir.Close()

	proc_files, err := dir.Readdirnames(0)

	if err != nil {
		log.Log("crit", "Could not read directory names from /proc: "+err.Error())
		return nil
	}

	all_pids := []string{}
	// XXX totally cheating here -- the only all-numeric filenames in this dir
	// will be pid directories. This should be faster than 4 bajillion stat
	// calls (that I'd have to do this to anyway).
	for _, fn := range proc_files {
		_, err := strconv.Atoi(fn)
		if err == nil {
			all_pids = append(all_pids, fn)
		}
	}

	for _, pid := range all_pids {
		path := "/proc/" + pid + "/cmdline"
		file, err := os.Open(path)

		if err != nil {
			log.Log("crit", "Could not open "+path+": "+err.Error())
			return nil
		}

		defer file.Close()

		cmdline, err := ioutil.ReadAll(file)

		if err != nil {
			log.Log("crit", "Could not read from "+path+": "+err.Error())
			return nil
		}

		cmdline_parts := strings.Split(string(cmdline), "\x00")

		if len(cmdline_parts) > 1 {
			cmdline_parts = cmdline_parts[0 : len(cmdline_parts)-1]
		}

		string_cmd := strings.Join(cmdline_parts, " ")

		if string_cmd == process {
			pids = append(pids, pid)
		}
	}

	return pids
}
