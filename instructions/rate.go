package instructions

import (
	"fmt"
	"time"
)

// Rates stores the information on how many hits per
// period of time any endpoint can accept
type Rate struct {
	// The counter would be incremeted to this value,
	// this can be used to count bytes for example
	Increment int64
	Value     int64
	Period    time.Duration
}

func NewRate(increment int64, value int64, period time.Duration) (*Rate, error) {
	if increment <= 0 {
		return nil, fmt.Errorf("increment should be > 0")
	}
	if value <= 0 {
		return nil, fmt.Errorf("Value should be > 0")
	}
	if period < time.Second || period > 24*time.Hour {
		return nil, fmt.Errorf("Period should be within [1 second, 24 hours]")
	}
	return &Rate{Increment: increment, Value: value, Period: period}, nil
}

// Calculates when this rate can be hit the next time from
// the given time t, assuming all the requests in the given
func (r *Rate) RetrySeconds(now time.Time) int {
	return int(r.NextBucket(now).Unix() - now.Unix())
}

//Returns epochSeconds rounded to the rate period
//e.g. minutes rate would return epoch seconds with seconds set to zero
//hourly rate would return epoch seconds with minutes and seconds set to zero
func (r *Rate) CurrentBucket(t time.Time) time.Time {
	return t.Truncate(r.Period)
}

// Returns the epoch seconds of the begining of the next time bucket
func (r *Rate) NextBucket(t time.Time) time.Time {
	return r.CurrentBucket(t.Add(r.Period))
}

// Returns the equivalent of the rate period in seconds
func (r *Rate) PeriodSeconds() int64 {
	return int64(time.Duration(r.Value) * time.Duration(r.Period) / time.Second)
}
