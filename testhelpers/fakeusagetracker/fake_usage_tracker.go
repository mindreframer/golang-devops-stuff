package fakeusagetracker

import (
	"time"
)

type FakeUsageTracker struct {
	DidStart bool

	UsageToReturn               float64
	MeasurementDurationToReturn time.Duration
}

func New() *FakeUsageTracker {
	return &FakeUsageTracker{}
}

func (tracker *FakeUsageTracker) StartTrackingUsage() {
	tracker.DidStart = true
}

func (tracker *FakeUsageTracker) MeasureUsage() (usage float64, measurementDuration time.Duration) {
	return tracker.UsageToReturn, tracker.MeasurementDurationToReturn
}
