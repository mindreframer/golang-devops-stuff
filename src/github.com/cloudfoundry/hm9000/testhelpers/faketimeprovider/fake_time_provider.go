package faketimeprovider

import "time"

type FakeTimeProvider struct {
	TimeToProvide time.Time
}

func (provider *FakeTimeProvider) Time() time.Time {
	return provider.TimeToProvide
}

func (provider *FakeTimeProvider) IncrementBySeconds(seconds uint64) {
	provider.TimeToProvide = time.Unix(provider.TimeToProvide.Unix()+int64(seconds), 0)
}
