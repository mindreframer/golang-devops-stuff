package fake_quota_manager

import (
	"sync"

	"github.com/vito/garden/backend"
)

type FakeQuotaManager struct {
	SetLimitsError error
	GetLimitsError error
	GetUsageError  error

	GetLimitsResult backend.DiskLimits
	GetUsageResult  backend.ContainerDiskStat

	MountPointResult string

	Limited map[uint32]backend.DiskLimits

	enabled bool

	sync.RWMutex
}

func New() *FakeQuotaManager {
	return &FakeQuotaManager{
		Limited: make(map[uint32]backend.DiskLimits),

		enabled: true,
	}
}

func (m *FakeQuotaManager) SetLimits(uid uint32, limits backend.DiskLimits) error {
	if m.SetLimitsError != nil {
		return m.SetLimitsError
	}

	m.Lock()
	defer m.Unlock()

	m.Limited[uid] = limits

	return nil
}

func (m *FakeQuotaManager) GetLimits(uid uint32) (backend.DiskLimits, error) {
	if m.GetLimitsError != nil {
		return backend.DiskLimits{}, m.GetLimitsError
	}

	m.RLock()
	defer m.RUnlock()

	return m.GetLimitsResult, nil
}

func (m *FakeQuotaManager) GetUsage(uid uint32) (backend.ContainerDiskStat, error) {
	if m.GetUsageError != nil {
		return backend.ContainerDiskStat{}, m.GetUsageError
	}

	m.RLock()
	defer m.RUnlock()

	return m.GetUsageResult, nil
}

func (m *FakeQuotaManager) MountPoint() string {
	return m.MountPointResult
}

func (m *FakeQuotaManager) Disable() {
	m.enabled = false
}

func (m *FakeQuotaManager) IsEnabled() bool {
	return m.enabled
}
