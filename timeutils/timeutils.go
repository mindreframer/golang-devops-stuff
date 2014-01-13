package timeutils

import (
	"fmt"
	"math"
	"math/rand"
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

type BackoffTimer struct {
	maxDelay float64
	minDelay float64
	Delay    float64
	retries  int
}

const factor = 2.7182818284590451
const jitter = 0.11962656472

func NewBackoffTimer(min float64, max float64) *BackoffTimer {
	bo := &BackoffTimer{minDelay: min, maxDelay: max, Delay: min}
	bo.bump(min)
	return bo
}

func (bo *BackoffTimer) bump(bound float64) {
	proposed := math.Min(bo.Delay*factor, bound)
	bo.Delay = rand.NormFloat64()*(proposed*jitter) + proposed
}

func (bo *BackoffTimer) Reset() {
	bo.bump(bo.minDelay)
}

func (bo *BackoffTimer) Increase() {
	bo.bump(bo.maxDelay)
}
