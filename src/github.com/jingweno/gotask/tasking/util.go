package tasking

import (
	"os"
	"os/exec"
)

func execCmd(input []string) error {
	name := input[0]
	args := input[1:]

	cmd := exec.Command(name, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}
