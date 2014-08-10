package process_count

import (
	gm "github.com/gollector/gollector_metrics"
	"logger"
)

func GetMetric(params interface{}, log *logger.Logger) interface{} {
	result, err := gm.ProcessCount(params.(string))

	if err != nil {
		log.Log("crit", err.Error())
		return nil
	}

	return result
}
