package analyzer

import (
	"math"
)

func ComputeCrashDelay(crashCount int, numberOfCrashesBeforeBackoffBegins int, startinDelay int, maximumDelay int) (delay int) {
	if crashCount < numberOfCrashesBeforeBackoffBegins {
		return 0
	}

	effectiveMaxCrashCount := int(math.Log2(float64(maximumDelay)/float64(startinDelay)) + float64(numberOfCrashesBeforeBackoffBegins))

	if crashCount > effectiveMaxCrashCount {
		return maximumDelay
	}

	return startinDelay * int(math.Pow(2.0, float64(crashCount-numberOfCrashesBeforeBackoffBegins)))
}
