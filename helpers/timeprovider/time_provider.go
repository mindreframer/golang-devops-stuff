package timeprovider

import "time"

type TimeProvider interface {
	Time() time.Time
	NewTickerChannel(name string, d time.Duration) <-chan time.Time
}

func NewTimeProvider() (provider *RealTimeProvider) {
	return &RealTimeProvider{}
}

type RealTimeProvider struct{}

func (provider *RealTimeProvider) Time() time.Time {
	return time.Now()
}

func (provider *RealTimeProvider) NewTickerChannel(name string, d time.Duration) <-chan time.Time {
	return time.NewTicker(d).C
}
