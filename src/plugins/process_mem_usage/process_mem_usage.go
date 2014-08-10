package process_mem_usage

import (
	gm "github.com/gollector/gollector_metrics"
	"logger"
)

func GetMetric(params interface{}, log *logger.Logger) interface{} {
	res, err := gm.ProcessMemoryUsage(params.(string))

	if err != nil {
		log.Log("crit", err.Error())
		return nil
	}

	return res
}
