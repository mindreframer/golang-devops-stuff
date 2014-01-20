package load_average

/*
#include <stdlib.h>
*/
import "C"

import (
	"logger"
)

func GetMetric(params interface{}, log *logger.Logger) interface{} {
	var loadavg [3]C.double

	log.Log("debug", "Calling getloadavg()")

	C.getloadavg(&loadavg[0], 3)

	return loadavg
}
