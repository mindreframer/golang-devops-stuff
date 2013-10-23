package timeprovider

import "time"

type TimeProvider interface {
	Time() time.Time
}

func NewTimeProvider() (provider *RealTimeProvider) {
	return &RealTimeProvider{}
}

type RealTimeProvider struct{}

func (provider *RealTimeProvider) Time() time.Time {
	return time.Now()
}
