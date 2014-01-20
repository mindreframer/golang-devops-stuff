package process_count

import (
	"logger"
	"util"
)

func GetMetric(params interface{}, log *logger.Logger) interface{} {
	return len(util.GetPids(params.(string), log))
}
