package record

import (
	"logger"
	"sync"
)

var recorded_metrics map[string]interface{}

var rwmutex sync.RWMutex

func GetMetric(params interface{}, log *logger.Logger) interface{} {
	endpoint := params.(string)

	rwmutex.RLock()
	result := recorded_metrics[endpoint]
	rwmutex.RUnlock()

	return result
}

func RecordMetric(name string, value interface{}, log *logger.Logger) {
	rwmutex.Lock()
	if recorded_metrics == nil {
		recorded_metrics = make(map[string]interface{})
	}
	recorded_metrics[name] = value
	rwmutex.Unlock()
}
