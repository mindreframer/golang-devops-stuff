package metricsaccountant

import (
	"time"

	"github.com/cloudfoundry/hm9000/models"
	"github.com/cloudfoundry/hm9000/store"
	"github.com/cloudfoundry/storeadapter"
)

type UsageTracker interface {
	StartTrackingUsage()
	MeasureUsage() (usage float64, measurementDuration time.Duration)
}

var startMetrics = map[models.PendingStartMessageReason]string{
	models.PendingStartMessageReasonCrashed:    "StartCrashed",
	models.PendingStartMessageReasonMissing:    "StartMissing",
	models.PendingStartMessageReasonEvacuating: "StartEvacuating",
}

var stopMetrics = map[models.PendingStopMessageReason]string{
	models.PendingStopMessageReasonDuplicate:          "StopDuplicate",
	models.PendingStopMessageReasonExtra:              "StopExtra",
	models.PendingStopMessageReasonEvacuationComplete: "StopEvacuationComplete",
}

type MetricsAccountant interface {
	TrackReceivedHeartbeats(metric int) error
	TrackSavedHeartbeats(metric int) error
	IncrementSentMessageMetrics(starts []models.PendingStartMessage, stops []models.PendingStopMessage) error
	TrackDesiredStateSyncTime(dt time.Duration) error
	TrackActualStateListenerStoreUsageFraction(usage float64) error
	GetMetrics() (map[string]float64, error)
}

type RealMetricsAccountant struct {
	store store.Store
}

func New(store store.Store) *RealMetricsAccountant {
	return &RealMetricsAccountant{
		store: store,
	}
}

func (m *RealMetricsAccountant) TrackReceivedHeartbeats(metric int) error {
	return m.store.SaveMetric("ReceivedHeartbeats", float64(metric))
}

func (m *RealMetricsAccountant) TrackSavedHeartbeats(metric int) error {
	return m.store.SaveMetric("SavedHeartbeats", float64(metric))
}

func (m *RealMetricsAccountant) TrackDesiredStateSyncTime(dt time.Duration) error {
	return m.store.SaveMetric("DesiredStateSyncTimeInMilliseconds", float64(dt)/float64(time.Millisecond))
}

func (m *RealMetricsAccountant) TrackActualStateListenerStoreUsageFraction(usage float64) error {
	return m.store.SaveMetric("ActualStateListenerStoreUsagePercentage", usage*100.0)
}

func (m *RealMetricsAccountant) IncrementSentMessageMetrics(starts []models.PendingStartMessage, stops []models.PendingStopMessage) error {
	metrics, err := m.GetMetrics()
	if err != nil {
		return err
	}

	for _, start := range starts {
		metrics[startMetrics[start.StartReason]] += 1
	}

	for _, stop := range stops {
		metrics[stopMetrics[stop.StopReason]] += 1
	}

	for key, value := range metrics {
		err := m.store.SaveMetric(key, float64(value))
		if err != nil {
			return err
		}
	}

	return nil
}

func (m *RealMetricsAccountant) GetMetrics() (map[string]float64, error) {
	metrics := map[string]float64{}
	for _, key := range startMetrics {
		metrics[key] = 0
	}
	for _, key := range stopMetrics {
		metrics[key] = 0
	}

	metrics["DesiredStateSyncTimeInMilliseconds"] = 0
	metrics["ActualStateListenerStoreUsagePercentage"] = 0
	metrics["SavedHeartbeats"] = 0
	metrics["ReceivedHeartbeats"] = 0

	for key := range metrics {
		value, err := m.store.GetMetric(key)
		if err == storeadapter.ErrorKeyNotFound {
			value = 0
		} else if err != nil {
			return map[string]float64{}, err
		}
		metrics[key] = value
	}

	return metrics, nil
}
