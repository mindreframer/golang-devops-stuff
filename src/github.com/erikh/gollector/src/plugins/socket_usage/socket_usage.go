package socket_usage

import (
	"io/ioutil"
	"logger"
	"os"
	"strings"
)

var SOCK_TYPES = []string{
	"tcp",
	"tcp6",
	"udp",
	"udp6",
	"udplite",
	"udplite6",
	"unix",
}

func GetMetric(params interface{}, log *logger.Logger) interface{} {
	sock_type := params.(string)

	found_sock_type := false

	for _, val := range SOCK_TYPES {
		if sock_type == val {
			found_sock_type = true
			break
		}
	}

	if !found_sock_type {
		log.Log("crit", "Invalid socket type: "+sock_type)
		return nil
	}

	f, err := os.Open("/proc/self/net/" + sock_type)

	if err != nil {
		log.Log("crit", "Could not open socket information for "+sock_type+": "+err.Error())
		return nil
	}

	defer f.Close()

	content, err := ioutil.ReadAll(f)

	if err != nil {
		log.Log("crit", "Trouble reading socket information for type "+sock_type+": "+err.Error())
		return nil
	}

	lines := strings.Split(string(content), "\n")
	return len(lines) - 1 // there's a one line header in these files
}
