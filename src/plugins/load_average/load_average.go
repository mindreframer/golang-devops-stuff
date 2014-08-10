package load_average

import (
	gm "github.com/gollector/gollector_metrics"
	"logger"
)

func GetMetric(params interface{}, log *logger.Logger) interface{} {
	return gm.LoadAverage()
}
