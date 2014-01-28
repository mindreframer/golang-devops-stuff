package agent

import (
	"fmt"
	"github.com/stripe-ctf/octopus/state"
)

func WorkingDir(i uint) string {
	return fmt.Sprintf("%s/node%d", state.Root(), i)
}

func ContainerWorkingDir(i uint) string {
	return fmt.Sprintf("%s/node%d", state.ContainerRoot(), i)
}

func NodeName(i uint) string {
	return fmt.Sprintf("node%d", i)
}

func SocketName(i uint) string {
	return "./" + NodeName(i) + ".sock"
}

func SocketPath(perspective, target uint) string {
	return WorkingDir(perspective) +
		"/" +
		SocketName(target)
}
