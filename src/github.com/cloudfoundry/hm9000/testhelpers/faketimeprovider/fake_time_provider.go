package faketimeprovider

import (
	"sync"

	"time"
)

type FakeTimeProvider struct {
	TimeToProvide time.Time

	ProvideFakeChannels bool

	RequestedTickerDurations map[string]time.Duration
	TickerChannels           map[string]chan time.Time
	tickerMutex              *sync.Mutex
}

func New(timeToProvide time.Time) *FakeTimeProvider {
	return &FakeTimeProvider{
		TimeToProvide:            timeToProvide,
		RequestedTickerDurations: map[string]time.Duration{},
		TickerChannels:           map[string]chan time.Time{},
		tickerMutex:              &sync.Mutex{},
	}
}

func (provider *FakeTimeProvider) Time() time.Time {
	return provider.TimeToProvide
}

func (provider *FakeTimeProvider) IncrementBySeconds(seconds uint64) {
	provider.TimeToProvide = time.Unix(provider.TimeToProvide.Unix()+int64(seconds), 0)
}

func (provider *FakeTimeProvider) NewTickerChannel(name string, d time.Duration) <-chan time.Time {
	if !provider.ProvideFakeChannels {
		return time.NewTicker(d).C
	}

	provider.tickerMutex.Lock()
	defer provider.tickerMutex.Unlock()

	provider.RequestedTickerDurations[name] = d
	provider.TickerChannels[name] = make(chan time.Time)

	return provider.TickerChannels[name]
}

func (provider *FakeTimeProvider) TickerChannelFor(name string) chan time.Time {
	provider.tickerMutex.Lock()
	defer provider.tickerMutex.Unlock()

	return provider.TickerChannels[name]
}

func (provider *FakeTimeProvider) TickerDurationFor(name string) time.Duration {
	provider.tickerMutex.Lock()
	defer provider.tickerMutex.Unlock()

	return provider.RequestedTickerDurations[name]
}
