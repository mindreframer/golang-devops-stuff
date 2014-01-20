package process_mem_usage

/*
#include <unistd.h>
unsigned int get_pgsz(void) {
  return sysconf(_SC_PAGESIZE);
}
*/
import "C"

import (
	"io/ioutil"
	"logger"
	"os"
	"strconv"
	"strings"
	"util"
)

func GetMetric(params interface{}, log *logger.Logger) interface{} {
	total := uint(0)
	page_size := uint(C.get_pgsz())

	for _, pid := range util.GetPids(params.(string), log) {
		path := "/proc/" + pid + "/statm"
		f, err := os.Open(path)

		if err != nil {
			log.Log("crit", "Could not open "+path+": "+err.Error())
			return nil
		}

		defer f.Close()

		content, err := ioutil.ReadAll(f)

		if err != nil {
			log.Log("crit", "Could not read from "+path+": "+err.Error())
			return nil
		}

		parts := strings.Split(string(content), " ")
		mem, err := strconv.Atoi(parts[1])

		if err != nil {
			log.Log("crit", "Trouble converting resident size "+parts[1]+" to integer: "+err.Error())
			return nil
		}

		total += uint(mem) * page_size
	}

	return total
}
