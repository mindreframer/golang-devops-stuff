package timeutils

import (
	"fmt"
	"time"
)

// Retuns the number of the day since epoch
func EpochDay(time time.Time) int64 {
	return time.Unix() / 86400
}

//Returns epoch seconds rounded to the given period
//e.g. minutes rate would return epoch seconds with seconds set to zero
//hourly rate would return epoch seconds with minutes and seconds set to zero
func RoundedBucket(t time.Time, p time.Duration) time.Time {
	return t.Truncate(p)
}

//This is a generic function that returns key composed of current time, key and period
func GetHit(now time.Time, key string, period time.Duration) string {
	return fmt.Sprintf(
		"%s_%s_%d", key, period.String(), RoundedBucket(now, period).Unix())
}
