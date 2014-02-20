package tachyon

import (
	"fmt"
	"reflect"
)

type Reporter interface {
	StartTasks(play *Play)
	FinishTasks(play *Play)
	StartHandlers(play *Play)
	FinishHandlers(play *Play)

	StartTask(task *Task, cmd Command, args string)
	FinishTask(task *Task, async bool)
}

type CLIReporter struct{}

var sCLIReporter *CLIReporter = &CLIReporter{}

func (c *CLIReporter) StartTasks(play *Play) {
	fmt.Printf("== tasks\n")
}

func (c *CLIReporter) FinishTasks(play *Play) {
	fmt.Printf("== Waiting on all tasks to finish...\n")
}

func (c *CLIReporter) StartHandlers(play *Play) {
	fmt.Printf("== Running any handlers\n")
}

func (c *CLIReporter) FinishHandlers(play *Play) {}

func (c *CLIReporter) StartTask(task *Task, cmd Command, args string) {
	if task.Async() {
		fmt.Printf("- %s &\n", task.Name())
	} else {
		fmt.Printf("- %s\n", task.Name())
	}

	if reflect.TypeOf(cmd).Elem().NumField() == 0 {
		fmt.Printf("  - %s: %s\n", task.Command(), args)
	} else {
		fmt.Printf("  - %#v\n  - %s: %s\n", cmd, task.Command(), args)
	}
}

func (c *CLIReporter) FinishTask(task *Task, async bool) {}
