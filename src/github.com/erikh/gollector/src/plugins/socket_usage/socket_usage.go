package socket_usage

import (
	gm "github.com/gollector/gollector_metrics"
	"logger"
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
	result, err := gm.SocketUsage(params.(string))

	if err != nil {
		log.Log("crit", err.Error())
		return nil
	}

	return result
}
