package fake_bandwidth_manager

import (
	"github.com/vito/garden/backend"
)

type FakeBandwidthManager struct {
	SetLimitsError error
	EnforcedLimits []backend.BandwidthLimits

	GetLimitsError  error
	GetLimitsResult backend.ContainerBandwidthStat
}

func New() *FakeBandwidthManager {
	return &FakeBandwidthManager{}
}

func (m *FakeBandwidthManager) SetLimits(limits backend.BandwidthLimits) error {
	if m.SetLimitsError != nil {
		return m.SetLimitsError
	}

	m.EnforcedLimits = append(m.EnforcedLimits, limits)

	return nil
}

func (m *FakeBandwidthManager) GetLimits() (backend.ContainerBandwidthStat, error) {
	if m.GetLimitsError != nil {
		return backend.ContainerBandwidthStat{}, m.GetLimitsError
	}

	return m.GetLimitsResult, nil
}
