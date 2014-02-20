package tachyon

import (
	"fmt"
)

func Main(args []string) int {
	if len(args) != 2 {
		fmt.Printf("Usage: tachyon <playbook>\n")
		return 1
	}

	playbook, err := LoadPlaybook(args[1])

	env := &Environment{}
	env.Init()

	err = playbook.Run(env)

	if err != nil {
		fmt.Printf("Error running playbook: %s\n", err)
		return 1
	}

	return 0
}
