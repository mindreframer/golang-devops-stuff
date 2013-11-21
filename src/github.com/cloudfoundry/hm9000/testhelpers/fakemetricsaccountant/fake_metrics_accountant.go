package fakemetricsaccountant

import (
	"github.com/cloudfoundry/hm9000/models"
	"time"
)

type FakeMetricsAccountant struct {
	IncrementSentMessageMetricsError error
	IncrementedStarts                []models.PendingStartMessage
	IncrementedStops                 []models.PendingStopMessage

	TrackedDesiredStateSyncTime                  time.Duration
	TrackedActualStateListenerStoreUsageFraction float64

	GetMetricsError   error
	GetMetricsMetrics map[string]float64

	ReceivedHeartbeats float64
	SavedHeartbeats    float64
}

func New() *FakeMetricsAccountant {
	return &FakeMetricsAccountant{
		IncrementedStarts: []models.PendingStartMessage{},
		IncrementedStops:  []models.PendingStopMessage{},

		GetMetricsMetrics: map[string]float64{},
	}
}

func (m *FakeMetricsAccountant) IncrementReceivedHeartbeats() error {
	m.ReceivedHeartbeats++
	return nil
}

func (m *FakeMetricsAccountant) IncrementSavedHeartbeats(by int) error {
	m.SavedHeartbeats += float64(by)
	return nil
}

func (m *FakeMetricsAccountant) IncrementSentMessageMetrics(starts []models.PendingStartMessage, stops []models.PendingStopMessage) error {
	m.IncrementedStarts = starts
	m.IncrementedStops = stops

	return m.IncrementSentMessageMetricsError
}

func (m *FakeMetricsAccountant) TrackDesiredStateSyncTime(dt time.Duration) error {
	m.TrackedDesiredStateSyncTime = dt
	return nil
}

func (m *FakeMetricsAccountant) TrackActualStateListenerStoreUsageFraction(usage float64) error {
	m.TrackedActualStateListenerStoreUsageFraction = usage
	return nil
}

func (m *FakeMetricsAccountant) GetMetrics() (map[string]float64, error) {
	return m.GetMetricsMetrics, m.GetMetricsError
}
