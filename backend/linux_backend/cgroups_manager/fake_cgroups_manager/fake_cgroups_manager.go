package fake_cgroups_manager

import (
	"path"
)

type FakeCgroupsManager struct {
	cgroupsPath string
	id          string

	SetError error

	setValues    []SetValue
	getCallbacks []GetCallback
	setCallbacks []SetCallback
}

type SetValue struct {
	Subsystem string
	Name      string
	Value     string
}

type GetCallback struct {
	Subsystem string
	Name      string
	Callback  func() (string, error)
}

type SetCallback struct {
	Subsystem string
	Name      string
	Callback  func() error
}

func New(cgroupsPath, id string) *FakeCgroupsManager {
	return &FakeCgroupsManager{
		cgroupsPath: cgroupsPath,
		id:          id,
	}
}

func (m *FakeCgroupsManager) Set(subsystem, name, value string) error {
	if m.SetError != nil {
		return m.SetError
	}

	for _, cb := range m.setCallbacks {
		if cb.Subsystem == subsystem && cb.Name == name {
			return cb.Callback()
		}
	}

	m.setValues = append(m.setValues, SetValue{subsystem, name, value})

	return nil
}

func (m *FakeCgroupsManager) Get(subsytem, name string) (string, error) {
	for _, cb := range m.getCallbacks {
		if cb.Subsystem == subsytem && cb.Name == name {
			return cb.Callback()
		}
	}

	for _, val := range m.setValues {
		if val.Subsystem == subsytem && val.Name == name {
			return val.Value, nil
		}
	}

	return "", nil
}

func (m *FakeCgroupsManager) SubsystemPath(subsystem string) string {
	return path.Join(m.cgroupsPath, subsystem, "instance-"+m.id)
}

func (m *FakeCgroupsManager) SetValues() []SetValue {
	return m.setValues
}

func (m *FakeCgroupsManager) WhenGetting(subsystem, name string, callback func() (string, error)) {
	m.getCallbacks = append(m.getCallbacks, GetCallback{subsystem, name, callback})
}

func (m *FakeCgroupsManager) WhenSetting(subsystem, name string, callback func() error) {
	m.setCallbacks = append(m.setCallbacks, SetCallback{subsystem, name, callback})
}
