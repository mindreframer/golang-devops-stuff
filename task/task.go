package task

import (
	"fmt"
	"github.com/codegangsta/cli"
	"github.com/jingweno/gotask/tasking"
)

type TaskSet struct {
	Name       string
	Dir        string
	PkgObj     string
	ImportPath string
	Tasks      []Task
}

func (ts *TaskSet) HasTasks() bool {
	return len(ts.Tasks) > 0
}

type Task struct {
	Name        string
	Usage       string
	Description string
	Flags       []Flag
	ActionName  string
	Action      func(*tasking.T)
}

func (t *Task) ToCLIFlags() (flags []cli.Flag) {
	for _, flag := range t.Flags {
		flags = append(flags, flag)
	}

	return
}

type Flag interface {
	cli.Flag
	DefType(importAsPkg string) string
}

type BoolFlag struct {
	cli.BoolFlag
}

func (f BoolFlag) getName() string {
	return f.Name
}

func (f BoolFlag) DefType(importAsPkg string) string {
	return fmt.Sprintf(`%s.NewBoolFlag("%s", "%s")`, importAsPkg, f.Name, f.Usage)
}

func NewBoolFlag(name, usage string) BoolFlag {
	return BoolFlag{cli.BoolFlag{Name: name, Usage: usage}}
}

type StringFlag struct {
	cli.StringFlag
}

func (f StringFlag) getName() string {
	return f.Name
}

func (f StringFlag) DefType(importAsPkg string) string {
	return fmt.Sprintf(`%s.NewStringFlag("%s", "%s", "%s")`, importAsPkg, f.Name, f.Value, f.Usage)
}

func NewStringFlag(name, value, usage string) StringFlag {
	return StringFlag{cli.StringFlag{Name: name, Value: value, Usage: usage}}
}
