package ratelimit

import (
	"fmt"
	"github.com/golang/glog"
	"github.com/mailgun/vulcan/backend"
	"github.com/mailgun/vulcan/command"
)

// Limits the requests based on stats stored in the backend, updates the stats
type RateLimiter interface {
	// This function checks if any of the rates is exceeded, in case if any of the rates is
	// exceeded, returns positive retry seconds specifying when it would make sense to retry.
	// Note that retry seconds does not guarantee that request will succeed after given seconds,
	// it just guarantiees that request would not succeed if tried before the given amount of seconds.
	// This is more a convenience for clients, so they can stop wasting excessive cycles.
	GetRetrySeconds(rates map[string][]*command.Rate) (retrySeconds int, err error)
	// Update stats within internal backend
	UpdateStats(requestBytes int64, rates map[string][]*command.Rate) error
}

type BasicRateLimiter struct {
	Backend backend.Backend
}

// Checks whether any of the given rates
func (rl *BasicRateLimiter) GetRetrySeconds(rates map[string][]*command.Rate) (retrySeconds int, err error) {
	for key, rateList := range rates {
		for _, rate := range rateList {
			counter, err := rl.Backend.GetCount(getKey(key, rate), rate.Period)
			if err != nil {
				return 0, err
			}
			if counter >= rate.Units {
				glog.Infof("Key('%s') %v is out of capacity", key, rate)
				return rate.RetrySeconds(rl.Backend.UtcNow()), nil
			}
		}
	}
	return 0, nil
}

func (rl *BasicRateLimiter) UpdateStats(requestBytes int64, rates map[string][]*command.Rate) error {
	for key, rateList := range rates {
		for _, rate := range rateList {
			err := rl.Backend.UpdateCount(getKey(key, rate), rate.Period, getCount(requestBytes, rate))
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func getCount(requestBytes int64, rate *command.Rate) int64 {
	switch rate.UnitType {
	case command.UnitTypeRequests:
		return 1
	case command.UnitTypeKilobytes:
		return requestBytes / 1024
	}
	return 1
}

func getKey(key string, rate *command.Rate) string {
	return fmt.Sprintf("%s_%s", key, command.UnitTypeToString(rate.UnitType))
}
