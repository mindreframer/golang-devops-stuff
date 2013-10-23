package faketimeprovider

import "time"

type FakeTimeProvider struct {
	TimeToProvide time.Time
}

func (provider *FakeTimeProvider) Time() time.Time {
	return provider.TimeToProvide
}
