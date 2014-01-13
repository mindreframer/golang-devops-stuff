package cgroups_manager

import (
	"bytes"
	"fmt"
	"os/exec"
	"path"
	"strings"

	"github.com/vito/garden/command_runner"
)

type ContainerCgroupsManager struct {
	cgroupsPath string
	containerID string

	runner command_runner.CommandRunner
}

func New(cgroupsPath, containerID string, runner command_runner.CommandRunner) *ContainerCgroupsManager {
	return &ContainerCgroupsManager{cgroupsPath, containerID, runner}
}

func (m *ContainerCgroupsManager) Set(subsystem, name, value string) error {
	return m.runner.Run(&exec.Cmd{
		Path: "bash",
		Args: []string{
			"-c",
			fmt.Sprintf("echo '%s' > %s", value, path.Join(m.SubsystemPath(subsystem), name)),
		},
	})
}

func (m *ContainerCgroupsManager) Get(subsystem, name string) (string, error) {
	catOut := new(bytes.Buffer)

	cmd := &exec.Cmd{
		Path:   "cat",
		Args:   []string{path.Join(m.SubsystemPath(subsystem), name)},
		Stdout: catOut,
	}

	err := m.runner.Run(cmd)
	if err != nil {
		return "", err
	}

	return strings.Trim(string(catOut.Bytes()), "\n"), nil
}

func (m *ContainerCgroupsManager) SubsystemPath(subsystem string) string {
	return path.Join(m.cgroupsPath, subsystem, "instance-"+m.containerID)
}
