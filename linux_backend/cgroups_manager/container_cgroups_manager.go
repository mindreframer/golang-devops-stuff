package cgroups_manager

import (
	"io/ioutil"
	"path"
	"strings"
)

type ContainerCgroupsManager struct {
	cgroupsPath string
	containerID string
}

func New(cgroupsPath, containerID string) *ContainerCgroupsManager {
	return &ContainerCgroupsManager{cgroupsPath, containerID}
}

func (m *ContainerCgroupsManager) Set(subsystem, name, value string) error {
	return ioutil.WriteFile(path.Join(m.SubsystemPath(subsystem), name), []byte(value), 0644)
}

func (m *ContainerCgroupsManager) Get(subsystem, name string) (string, error) {
	body, err := ioutil.ReadFile(path.Join(m.SubsystemPath(subsystem), name))
	if err != nil {
		return "", err
	}

	return strings.Trim(string(body), "\n"), nil
}

func (m *ContainerCgroupsManager) SubsystemPath(subsystem string) string {
	return path.Join(m.cgroupsPath, subsystem, "instance-"+m.containerID)
}
